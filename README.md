# PodSpider
Crawl all pod metrics you need

THIS PROJECT IS STILL WIPWIPWIPWIP

## Build
```shell
cd ./podspider
go build
```

## Run
```
go run ./main.go
```
Typical usage:
* Setup the configuration of podspider
* Run this software
Then the software will start dumping your pods information, currently includes `cpu usage`, `memory usage`, `network usage` and `disk usage`.

To control what you want to dump, please check the `ResourceManager.Tick()`

## Roadmap
[ ] Client-server mode for distributed cgroup manipulation
[ ] Dynamic resource allocation
