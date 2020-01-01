
### version
```
v1.1.0
```


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


### usage
```php
<?php
$redis_handle = new Redis();
$redis_handle->connect('127.0.0.1', 6788);
echo $redis_handle->rawCommand("token", "zybx") . PHP_EOL;
echo $redis_handle->rawCommand("ticket", "zybx") . PHP_EOL;
echo $redis_handle->rawCommand("token", "zybx", 1) . PHP_EOL;
echo $redis_handle->rawCommand("ticket", "zybx", 1) . PHP_EOL;
```