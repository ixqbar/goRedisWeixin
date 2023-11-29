
### version
```
v1.4.1
```

### 配置
```
[DEFAULT]
;v1.2.0之前版本
;address=0.0.0.0:6788

;v1.2.0版本
web=0.0.0.0:6780
redis=0.0.0.0:6788
data_file=/data/server/weixin/conf/data.dat

;获取别名
[zybx]
app_id=
app_secret=
is_enterprise=0
```
* v1.2.0版本后配置address废弃，增加配置web,redis服务区分
* v1.3.0版本后增加是否企业微信标记is_enterprise
* v1.4.0版本后增加zall,ztoken,zticket

### token ticket 命令
```
token zybx
ticket zybx
ztoken zybx
zticket zybx
zall zybx

强制重刷
token zybx 1   
ticket zybx 1
ztoken zybx 1
zticket zybx 1

保存
save
```
* zybx 为对应 ini配置section名称，可与公众号对应

### 使用

#### http
```
curl 'http://127.0.0.1:6780/token/zybx/'
curl 'http://127.0.0.1:6780/token/zybx/1'
curl 'http://127.0.0.1:6780/ticket/zybx/'
curl 'http://127.0.0.1:6780/ticket/zybx/1'
curl 'http://127.0.0.1:6780/zticket/zybx/'
curl 'http://127.0.0.1:6780/zticket/zybx/1'
curl 'http://127.0.0.1:6780/ztoken/zybx/'
curl 'http://127.0.0.1:6780/ztoken/zybx/1'
curl 'http://127.0.0.1:6780/zall/zybx'
```

#### redis
```php
<?php
$redis_handle = new Redis();
$redis_handle->connect('127.0.0.1', 6788);
echo $redis_handle->rawCommand("token", "zybx") . PHP_EOL;
echo $redis_handle->rawCommand("ticket", "zybx") . PHP_EOL;
echo $redis_handle->rawCommand("token", "zybx", 1) . PHP_EOL;
echo $redis_handle->rawCommand("ticket", "zybx", 1) . PHP_EOL;
```
