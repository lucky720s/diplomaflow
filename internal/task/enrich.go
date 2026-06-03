package task

import (
	"context"
	"strings"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	taskv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/task/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

// Обогащение UserPreview именами/email из auth_service.
//
// Конвертеры (taskToProto и т.п.) заполняют только Id; здесь мы батчем
// подтягиваем данные пользователей и дописываем FullName/Email в готовый ответ.
// Обогащение best-effort: если auth недоступен — отдаём ответ как есть (с id),
// чтобы не ронять основной запрос.
//
// Замечание: auth.UserPreview не содержит avatar_url, поэтому AvatarUrl остаётся
// пустым до появления аватаров в auth_service.

// resolveUsers батчем получает превью пользователей по id.
func (h *Handler) resolveUsers(ctx context.Context, ids []int64) map[int64]*authv1.UserPreview {
	out := make(map[int64]*authv1.UserPreview)
	if h.authClient == nil || len(ids) == 0 {
		return out
	}

	seen := make(map[int64]struct{}, len(ids))
	uniq := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return out
	}

	ictx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "task_service")
	resp, err := h.authClient.BatchGetUserPreviews(ictx, &authv1.BatchGetUserPreviewsRequest{Ids: uniq})
	if err != nil {
		h.logger.Warn("user enrichment failed", zap.Error(err))
		return out
	}
	for _, u := range resp.GetUsers() {
		out[u.GetId()] = u
	}
	return out
}

func fillUserPreview(p *taskv1.UserPreview, u *authv1.UserPreview) {
	if p == nil || u == nil {
		return
	}
	full := strings.TrimSpace(u.GetFirstName() + " " + u.GetLastName())
	if full != "" {
		p.FullName = full
	}
	p.Email = u.GetEmail()
}

// collectFromTasks собирает id пользователей из задач (assignee + created_by).
func collectFromTasks(tasks []*taskv1.Task, ids *[]int64) {
	for _, t := range tasks {
		if t.GetAssignee() != nil {
			*ids = append(*ids, t.Assignee.Id)
		}
		if t.GetCreatedBy() != nil {
			*ids = append(*ids, t.CreatedBy.Id)
		}
	}
}

func applyToTasks(tasks []*taskv1.Task, users map[int64]*authv1.UserPreview) {
	for _, t := range tasks {
		if t.GetAssignee() != nil {
			fillUserPreview(t.Assignee, users[t.Assignee.Id])
		}
		if t.GetCreatedBy() != nil {
			fillUserPreview(t.CreatedBy, users[t.CreatedBy.Id])
		}
	}
}

// enrichTasks обогащает список задач (assignee/created_by).
func (h *Handler) enrichTasks(ctx context.Context, tasks []*taskv1.Task) {
	if len(tasks) == 0 {
		return
	}
	var ids []int64
	collectFromTasks(tasks, &ids)
	applyToTasks(tasks, h.resolveUsers(ctx, ids))
}

func (h *Handler) enrichComments(ctx context.Context, comments []*taskv1.Comment) {
	if len(comments) == 0 {
		return
	}
	var ids []int64
	for _, c := range comments {
		if c.GetAuthor() != nil {
			ids = append(ids, c.Author.Id)
		}
	}
	users := h.resolveUsers(ctx, ids)
	for _, c := range comments {
		if c.GetAuthor() != nil {
			fillUserPreview(c.Author, users[c.Author.Id])
		}
	}
}

func (h *Handler) enrichAttachments(ctx context.Context, atts []*taskv1.Attachment) {
	if len(atts) == 0 {
		return
	}
	var ids []int64
	for _, a := range atts {
		if a.GetUploadedBy() != nil {
			ids = append(ids, a.UploadedBy.Id)
		}
	}
	users := h.resolveUsers(ctx, ids)
	for _, a := range atts {
		if a.GetUploadedBy() != nil {
			fillUserPreview(a.UploadedBy, users[a.UploadedBy.Id])
		}
	}
}

func (h *Handler) enrichActivities(ctx context.Context, acts []*taskv1.ActivityLogEntry) {
	if len(acts) == 0 {
		return
	}
	var ids []int64
	for _, a := range acts {
		if a.GetActor() != nil {
			ids = append(ids, a.Actor.Id)
		}
	}
	users := h.resolveUsers(ctx, ids)
	for _, a := range acts {
		if a.GetActor() != nil {
			fillUserPreview(a.Actor, users[a.Actor.Id])
		}
	}
}

// enrichGetTask обогащает агрегированный ответ GetTask одним батчем по всем
// упомянутым пользователям (задача + комментарии + вложения + активность + watchers).
func (h *Handler) enrichGetTask(ctx context.Context, resp *taskv1.GetTaskResponse) {
	if resp == nil {
		return
	}
	var ids []int64
	if resp.Task != nil {
		collectFromTasks([]*taskv1.Task{resp.Task}, &ids)
	}
	for _, c := range resp.RecentComments {
		if c.GetAuthor() != nil {
			ids = append(ids, c.Author.Id)
		}
	}
	for _, a := range resp.Attachments {
		if a.GetUploadedBy() != nil {
			ids = append(ids, a.UploadedBy.Id)
		}
	}
	for _, a := range resp.RecentActivity {
		if a.GetActor() != nil {
			ids = append(ids, a.Actor.Id)
		}
	}
	for _, w := range resp.Watchers {
		if w != nil {
			ids = append(ids, w.Id)
		}
	}

	users := h.resolveUsers(ctx, ids)
	if resp.Task != nil {
		applyToTasks([]*taskv1.Task{resp.Task}, users)
	}
	for _, c := range resp.RecentComments {
		if c.GetAuthor() != nil {
			fillUserPreview(c.Author, users[c.Author.Id])
		}
	}
	for _, a := range resp.Attachments {
		if a.GetUploadedBy() != nil {
			fillUserPreview(a.UploadedBy, users[a.UploadedBy.Id])
		}
	}
	for _, a := range resp.RecentActivity {
		if a.GetActor() != nil {
			fillUserPreview(a.Actor, users[a.Actor.Id])
		}
	}
	for _, w := range resp.Watchers {
		fillUserPreview(w, users[w.GetId()])
	}
}
