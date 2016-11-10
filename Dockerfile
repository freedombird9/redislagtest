FROM golang
MAINTAINER Yongfeng Zhang <yongfezh@cisco.com> 
ADD . /go/src/redislagtest
WORKDIR /go/src/redislagtest
RUN make deps
RUN make install
WORKDIR /go/bin
EXPOSE 8080
CMD ["bash"]
