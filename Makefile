.PHONY: wire proto

wire:
	@echo "Generating Wire..."
	cd internal/auth && wire
	cd internal/role && wire
	cd internal/university && wire
	cd internal/workflow && wire
	cd internal/gateway && wire
	@echo "Wire generated!"

proto:
	powershell -ExecutionPolicy Bypass -File scripts/gen_proto.ps1
