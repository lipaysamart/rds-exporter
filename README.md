## Deploy

### Docker-compose

Pre-requisites:

- Docker
- docker-compose

```bash
# copy and modify example.yaml first
cd deploy
docker-compose up -d
```
### Kubernetes

Pre-requisites:

- Kubernetes
```
k apply -f deploy/rds-deploy.yaml
k get po -n monitoring  -w
```

## Prometheus
```
- job_name: 'rds-exporter'
  scrape_interval: 60s
  scrape_timeout: 60s
  static_configs:
  - targets: ['rds-exporter:9527']
    labels:
      account_name: xxxx
      provider: aliyun # or aliyun_jst
```

## UseLimit
* 一个 exporter 实例只能抓取一个地域中的指标 

## Ref
* https://github.com/fengxsong/aliyun-exporter --基于这位大佬的仓库，做了些修改及一些弃用的包更新