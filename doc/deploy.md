
# nginx + docker
自行安装相关环境，省事直接用了宝塔面板

docker-compose.yml
```
version: '3.7'
services: 
  opencatd:
    image: mirrors2/opencatd-open
    container_name: opencatd-open 
    restart: unless-stopped
    ports:
      - 8088:80
    volumes:
      - $PWD/db:/app/db
```
nginx配置
```
location /
{
    proxy_pass http://localhost:8088;
    proxy_set_header Host localhost;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header REMOTE-HOST $remote_addr;
}
```