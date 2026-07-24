run:
	go run cmd/yoga/main.go

lint:
	golangci-lint run ./...

test:
	go test -v ./...

build:
	go build -v -o bin/yoga cmd/yoga/main.go

mocks:
	rm -rf mock/
	mockgen --source=internal/domain/repositories/repositories.go --destination=mock/repositories.go --package=mock
	mockgen --source=internal/domain/repositories/unit_of_work.go --destination=mock/unit_of_work.go --package=mock
	mockgen --source=internal/application/bookings/contract.go --destination=mock/services.go --package=mock
	mockgen --source=internal/application/classes/contract.go --destination=mock/services.go --package=mock
	mockgen --source=internal/application/location/contract.go --destination=mock/location_link_provider.go --package=mock
	mockgen --source=internal/application/passes/contract.go --destination=mock/services.go --package=mock
	mockgen --source=internal/application/pendingbookings/contract.go --destination=mock/services.go --package=mock
	mockgen --source=internal/domain/notifier/notifier.go --destination=mock/notifier.go --package=mock
