1：杀死项目

pkill -9 -f tesla-server

2：查看项目运行情况

ps aux | grep tesla

3：编译项目（更新文件需要重新编译）


go build -o tesla-server ./cmd/

4：启动项目
./tesla-server
