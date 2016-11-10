BUILD_NUMBER?=latest

install:
	go install redislagtest/...

deps:
	-cd $(GOPATH)/src; \
	if [ ! -d "go-jasperlib" ]; then git clone http://qa1-sjc002-030.i.jasperwireless.com/cc/go-jasperlib.git; fi
	-go get -t -u -f -insecure redislagtest/...

docker:
	docker build  -t redislagtest:$(BUILD_NUMBER) .

FORCE:
.PHONY: deps docker
