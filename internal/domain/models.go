package domain

type University struct {
	ID   string
	Name string
}

type Department struct {
	ID           string
	Name         string
	UniversityID string
}

type Student struct {
	ID           string
	FullName     string
	DepartmentID string
}

type DiplomaProject struct {
	ID        string
	Title     string
	StudentID string
	Status    string // draft, approved, defended
}
