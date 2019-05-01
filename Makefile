all : build
run : 
	go run nhdl.go
build : 
	go build nhdl.go
dep :
	go get -u github.com/gocolly/colly/...