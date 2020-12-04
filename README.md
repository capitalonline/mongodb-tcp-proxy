## Install & Run
**System requirements:**

**On a Linux host:** docker 17.06.0-ce+ and docker-compose 1.18.0+ .
## Install
  docker build -t="mongodb-tcp-proxy:version" .
## Run
  docker run -d --name mongodb-tcp-proxy -v /app/logs:/app/logs -p 8474:8474 mongodb-tcp-proxy:version

## Build 
**System requirements:** 
**go version: 1.14+**
## Linux 
**linux to windows**
```    
       SET CGO_ENABLED=0  
       SET GOOS=windows
       SET GOARCH=amd64
       cd cmd && go build -o mongodb-proxy
```

## Windows
**windows to linux**
```    
       SET CGO_ENABLED=0  
       SET GOOS=linux
       SET GOARCH=amd64
       cd cmd && go build -o mongodb-proxy
       #go build -gcflags=-trimpath=${GOPATH} -asmflags=-trimpath=${GOPATH} -o mongodb-proxy
```



   
