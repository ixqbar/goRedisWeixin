
### version
```
v1.2.0
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
```
* v1.2.0版本后配置有变更

### token ticket 命令
```
token zybx
ticket zybx

强制重刷
token zybx 1   
ticket zybx 1

保存
save
```
* zybx 为对应 ini配置section名称，可与公众号对应

### 使用

#### http
```
curl 'http://127.0.0.1:6780/token/boc/'
curl 'http://127.0.0.1:6780/token/boc/1'
curl 'http://127.0.0.1:6780/ticket/boc/'
curl 'http://127.0.0.1:6780/ticket/boc/1'
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
