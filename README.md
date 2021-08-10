# Ultraviolet - Alpha v0.12.1

## What is Ultraviolet?
Its a reverse minecraft proxy, capable of serving as a placeholder when the server is offline for status response to clients.   

one could also say [infrared](https://github.com/haveachin/infrared) but different.

Everything which ultraviolet has or does right now is not final its possible that it will change how it currently works therefore the reason its still in Alpha. If wanna complain that something has been changed and/or it broke something thats your own fault, its in Alpha version after all. 

Thinks likely to change: 
- Run command
- Prometheus support
- config file(s) (structure of the files themselves)
- config flag


## Features
[x] Proxy Protocol(v2) support  
[x] RealIP (v2.4&v2.5)  
[x] Rate limiting -> Login verification  
[x] Status caching (online status only)  
[x] Offline status placeholder  
[x] Prometheus Support  
... More coming later?

## Some notes
### Limited connections when running binary
Because linux the default settings for fd is 1024, this means that you can by default Ultraviolet can have 1024 open connections before it starts refusing connections because it cant open anymore fds. Because of some internal queues you should consider increasing the limit if you expect to proxy over 900 open connections at the same time. 

### How to build
Ultraviolet can be ran by using docker or you can also build a binary yourself by running:
```
$ cd cmd/Ultraviolet/
$ go build
```  

### How to run

Ultraviolet will when no config is specified by the command use `/etc/ultraviolet` as work dir and create here an `ultraviolet.json` file for you.
```
$ ./ultraviolet run
```  

### Tableflip
This has implemented [tableflip](https://github.com/cloudflare/tableflip) which should make it able to reload/hotswap Ultraviolet without closing existing connections on Linux and macOS. Ultraviolet should still be usable on windows (testing purposes only pls). 
Check their [documentation](https://pkg.go.dev/github.com/cloudflare/tableflip) to know what or how. 

IMPORTANT: There is a limit of one 'parent' process. So when you reload Ultraviolet once you need to wait until the parent process is closed (all previous connections have been closed) before you can reload it again.

## Command-Line 
The follows commands can be used with ultraviolet, all flags (if related) should work for every command and be used by every command if you used it for one command.

So far it only can use:
- run
- reload

### Flags
`-configs` specifies the path to the config directory [default: `"/etc/ultraviolet/"`]  


## How does some stuff work
### rate limiting
~~With rate limiting Ultraviolet will allow a specific number of connections to be made to the backend within a given time frame. It will reset when the time frame `rateCooldown` has passed. When the number has been exceeded but the cooldown isnt over yet, Ultraviolet will behave like the server is offline.  
By default status request arent rate limited but you can turn this on. When its turned on and the connection rate exceeds the rate limit it can still send the status of the to the player when cache status is turned on. 
Disabling rate limiting can be done by setting it to 0 and allows as many connections as it can to be created. (There is no difference in rate limiting disconnect and offline disconnect packets yet.)~~
It works differently now, based on playernames and ips instead of absolute connections. Status request will never be blocked, if you want to prevent spam from those turn on the status caching. (Later more detail explanation)  

### state update Cooldown
To prevent a lot of calls being made to the backend without a reason Ultraviolet will keep track of the state from the backend. The state is currently being based on whether or not the backend will accept an tcp connection or not. When this happened and ultraviolet knows that the backend is `ONLINE` or `OFFLINE` it will wait the time `stateUpdateCooldown` is set to before it will check the state of the backend again. 

### RealIP 2.5
This version of RealIP uses a public and private key to sign the handshake to ensure the backend that it came from a valid proxy. We dont have the private key which can be used with the public key which is included as default in the RealIP plugin since version 2.5. For that reason you need to replace the public key inside the plugin with the public key generated by Ultraviolet or with the one you generated yourself. The only format which can be used is an ECDSA with SHA512. 

Its possible to replace the key without rebuilding the RealIP plugin itself. You can do this in different way for example with 7zip archive GUI or with a terminal command. For example you could use:
```
jar -uf path/to/jarfile -C path/to/dir/in/jar path/to/file 
```

## Config
- Time config values are based on go's duration formatting, valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". They can be used in combination with each other "1m30s".
- All config values left blank will result into their default value. For example if you dont have `"rateLimit": 5` inside your json, it will automatically put it on 0 which will also disable ratelimiting.  
- Inside the `examples` folder there is example of a server config file and the ultraviolet config file. 
- If its a place where you can use an ipv4, ipv6 should also work as well. Not specifying an ip and only using `:25565` will/might end up using either or both. 
- The main config file needs to have the name `ultraviolet.json`.  
- Every server config needs to end with `.json`. Server config files will be searched for recursively.

### Ultraviolet Config
|Field name|Default | Description| 
|:---:|:---:|:---|
|listenTo|""|The address Ultraviolet will listen to for receiving connections.|
|defaultStatus|[this](#status-config-value)|The status Ultraviolet will send to callers when it receives a status handshake where the server address header isnt recognized.|
|numberOfWorkers|10|The number of 'workers' (code running in their own goroutine) Ultraviolet will have running, 0 will disabled it. This means that it wont process any incomming connections. |
|numberOfListeners|1|The number of listeners Ultraviolet will have running, 0 will disabled which means that it wont accept any incomming connections. |
|acceptProxyProtocol|false|If set to through all connections will be viewed as proxy protocol connections if it doesnt receive the header the connections will be closed. |
|enablePrometheus|true|This will enable the prometheus endpoint.|
|prometheusBind|":9100"|Here you can let it know to which address it should listen.|
|apiBind|"127.0.0.1:9099"|The address the website will listen for its api usage.|

### Server Config
|Field name|Default | Description| 
|:---:|:---:|:---|
|domains|[""]|Place in here all urls which should be used by clients to target the backend.|
|proxyTo|""|It will call this ip/url when its creating a connection to the server.|
|proxyBind|""|The ip it should be using while connection to the backend. If it cant use the given value it will fail and the connection wont be created.|
|dialTimeout|"1s"|Timeout is the maximum amount of time a dial will wait for a connect to complete.|
|useRealIPv2.4|false|RealIP will only be used when players want to login. If both are turned on, it will use v2.4.|
|useRealIPv2.5|false|RealIP will only be used when players want to login. If both are turned on, it will use v2.4. If there isnt a key in the path, it will generate a key for you, the file of the key will begin with the first domain of this backend config.|
|realIPKeyPath|""|The path of the private key which will be used to encrypt the signature. Its not checking for file permissions or anything like that.|
|sendProxyProtocol|false|Whether or not it should send a ProxyProtocolv2 header to the target.|
|disconnectMessage|""|The message a user will get when its tries to connect to a offline server|
|offlineStatus|[this](#status-config-value)|The status it will send the player when the server is offline.|
|rateLimit|0|The number of connections it will allow to be made to the backend in the given `rateCooldown` time. 0 will disable rate limiting.|
|rateCooldown|"1s"|rateCooldown is the time which it will take before the rateLimit will be reset.|
|banListCooldown|"5m"|The amount of time someone an ip will need to wait (when its banned) before it will get unbanned from joining a specific server.|
|reconnectMsg|"Please reconnect to verify yourself"|The message the player will be shown when its trying to join a server which is above its rate limit.|
|stateUpdateCooldown|"1s"|The time it will assume that the state of the server isnt changed (that server isnt offline now while it was online the last time we checked). |
|cacheStatus|false|Turn on or off whether it should cache the online cache of the server. If the server is recognized as `OFFLINE` it will send the offline status to the player.|
|validProtocol|0|validProtocol is the protocol integer the handshake will have when sending the handshake to the backend. Its only necessary to have this when `cacheStatus` is on.|
|cacheUpdateCooldown|"1s"|The time it will assume that the statys of the server isnt changed (including player count). |


### Status Config value
A status config is build with the following fields
|Field name|Default | Description| 
|:---:|:---:|:---|
|name|""|This is the 'name' of the status response. Its the text which will appear on the left of the latency bar.|
|protocol|0|This is the protocol it will use. For more information about it or to see what numbers belong to which versions check [this website](https://wiki.vg/Protocol_version_numbers) |
|text|""|This is also known as the motd of server.|
|favicon|""|This is the picture it will send to the player. If you want to use this turn the picture you wanna use into a base64 encoded string.|


