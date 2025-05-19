APP := fastdns
APP_PKG := github.com/dwisiswant0/$(APP)

.PHONY: vet
vet:
	go vet ./...

.PHONY: test-build
test-build:
	sudo env "PATH=$(PATH)" go test -c github.com/dwisiswant0/fastdns

.PHONY: test
test:
	sudo ./$(APP).test -test.v github.com/dwisiswant0/fastdns

.PHONY: bench
bench:
	sudo ./$(APP).test -test.benchmem -test.cpuprofile=cpu.prof -test.memprofile=mem.prof -test.run=^$$ -test.bench=. github.com/dwisiswant0/fastdns
