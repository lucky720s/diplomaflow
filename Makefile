.PHONY: wire proto

wire:
	@echo "Generating Wire..."
	cd internal/auth && wire
	cd internal/role && wire
	cd internal/university && wire
	cd internal/workflow && wire
	cd internal/gateway && wire
	cd internal/file && wire
	cd internal/form && wire
	cd internal/team && wire
	cd internal/notification && wire
	@echo "Wire generated!"

proto:
	powershell -ExecutionPolicy Bypass -File scripts/gen_proto.ps1
