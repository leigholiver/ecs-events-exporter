# ecs-events-exporter
tool to scrape [ECS service event messages](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-event-messages.html) and push them into grafana loki

## log labels
the following labels are attached to loki logs:
- `role_arn` the role arn used to scrape the logs
- `aws_account` the aws account ID the logs originated from
- `aws_region` the aws region the logs originated from
- `ecs_cluster` the ecs cluster the logs originated from
- `service_name` the ecs service the logs originated from

in addition, any tags associated with the ECS service will be sent as log labels, in the format `tag_<key>=<value>`

## configuration
### via env vars
env vars take precedence over the config file
- `CONFIG_FILE` the path to a config file to load (default `config.yaml`)
- `SCAN_INTERVAL` how often to scrape logs in seconds (default `60`)
- `LOKI_URL` the loki url to push to, in the format `http://example.com/loki/api/v1/push`
- `LOKI_ORG_ID` the org id to set on loki logs

### via config file
```yaml
# config.yaml

# how often to scrape ECS for logs
scan_interval: 60 # in seconds

logging:
  logger: loki # or "stdout"
  options:
    # loki url to push to
    url: http://localhost:3102/loki/api/v1/push
    # loki org id
    org_id: tenant1

# by default we use any accessible ambient credentials
# set to true to disable this behaviour
ignore_default_credentials: true

# regions to check for ECS clusters
# this can be overriden by assumed roles, see below
regions:
  - eu-west-1

# iam roles to assume for multi-account log scraping
roles:
  # this role will use the default region above
  - role_arn: arn:aws:iam::987654321098:role/AnotherRole

  # this role will scrape in us-west-1
  - role_arn: arn:aws:iam::123456789012:role/RoleName
    regions:
      - us-west-1


# by default we scrape all visible ECS clusters, but this can be filtered:

# a list of specific ECS cluster names to scrape
clusters:
  - my-cluster
  - my-other-cluster

# a list of tag-based filters to apply to clusters
# if specified alongside specific cluster names, this will
# be used to filter within the provided list
cluster_tags:
  # all clusters tagged Environment=Production
  - Environment: Production
  # or, all clusters tagged with Application=MyApp AND Environment=Staging
  - Application: MyApp
    Environment: Staging


# by default we scrape all visible ECS services, but this can be filtered:

# a list of specific ECS service names to scrape
services:
  - my-service
  - my-other-service

# a list of tag-based filters to apply to services
# if specified alongside specific service names, this will
# be used to filter within the provided list
service_tags:
  # all services tagged Environment=Production
  - Environment: Production
  # or, all services tagged with Application=MyApp AND Environment=Staging
  - Application: MyApp
    Environment: Staging
```

## development
- Run: `go run .`
- Build: `go build .`

the `test-support/` directory has a preconfigured loki/grafana docker-compose stack to test against
- `docker-compose up`
- grafana can be accessed at http://localhost:3000
- the loki api is accessible at http://localhost:3100/loki/api/v1/push
- the org-id must be set to `tenant1`
