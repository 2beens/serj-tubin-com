<div align="center">

  # serj-tubin.com
  ### experimentation sandbox app
  ![Actions Status](https://github.com/2beens/serj-tubin-com/workflows/CI/badge.svg)
  ![Actions Status](https://github.com/2beens/serj-tubin-com/workflows/CodeQL/badge.svg)
  ![Go Report Card](https://goreportcard.com/badge/github.com/2beens/serj-tubin-com)
  ![Go VulnCheck](https://github.com/2beens/serj-tubin-com/workflows/VulnCheck/badge.svg?branch=master)
  
  Personal tech sandbox and testing site 🕵️‍♀️
  
  Hosted and available at: https://www.serj-tubin.com/
  
  (also at https://2beens.xyz 🤷🏼‍)

</div>

### Used tech 🛠
* Golang 🦫🩵to make this all happen 👨🏼‍💻
* `Dockerfile` and `docker/` folder contain configs for running the service stack within Docker
  * `cd docker && docker-compose up -d` to run it all
* Aerospike to store those visitor board messages 💿
    * there are better choices than Aerospike, I know, just wanted to test Aerospike
    * update: I am thinking to replace it for something else
    * update apr 2023: aerospike killed, using postgresql for visitor board messages
* Tried to use Elasticsearch Stack for logging and monitoring on a different server, but it was kinda hungry for memory, and exceeded the needs of this humble project (will try something else lighter)
  * 2023: Honeycomb, Prometheus/Grafana and Sentry are used now
* Used CloudFlare to put everything behind it, but only paid plans allowed passing of real client ip in the proxied requests
* ~~https://freegeoip.app~~ since 2022, ~~https://ipbase.com~~, dec 22: https://ipinfo.io
    * used for geo IP data
* http://api.openweathermap.org
    * used for weather data
* GitHub Workflow Actions for a part of CI/CD
    * unit testing
    * static code analysis
    * deploy on new release
* PostgreSQL to store blog posts and personal web history (netlog), notes, etc.
* Redis to store session data
* Prometheus is for metrics (then Grafana to visualize them)
* Honeycomb is for distributed tracing
* Sentry for error tracking and alerting
* VueJS (with Vuetify) to make the frontend part (was my first Vue project, so I don't like it that much)
    * source @ https://github.com/2beens/serj-tubin-vue
* ❗️ Disclaimer: some parts of the system are deliberately unoptimized or complicated for testing ☑️ / learning 👨🏼‍🏫 / trial 🛠 purposes
  * Moreover, some parts are quite old now and thus not very go idiomatic

### TODO: Observability (done ✅)
- use otel to collect and send metrics and tracing data
  - for metrics use Prometheus
  - for tracing use free Honeycomb plan
    - https://ui.honeycomb.io/serj-tubin-com/environments/test/send-data#

### Easiest way to run it: yes, Docker 🐳🐳🐳🐳
```sh
cd docker
make up
# or make up-win on windows
```
