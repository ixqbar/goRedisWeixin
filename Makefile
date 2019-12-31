TARGET=goRedisWeixin

all: linux mac win

linux: 
	GOOS=linux GOARCH=amd64 go build -o ./bin/${TARGET}_${@} ./src/weixin/

mac: 
	GOOS=darwin GOARCH=amd64 go build -o ./bin/${TARGET}_${@} ./src/weixin/

win:
	GOOS=windows GOARCH=amd64 go build -o ./bin/${TARGET}.exe ./src/weixin/
	GOOS=windows GOARCH=386 go build -o ./bin/${TARGET}-i386.exe ./src/weixin/

clean:
	rm -rf ./bin/${TARGET}_*	
