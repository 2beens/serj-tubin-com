<div align="center">

  # serj-tubin.com
  ### experimentation sandbox app
  [![Actions Status](https://github.com/2beens/serj-tubin-com/workflows/CI/badge.svg)](https://github.com/2beens/serj-tubin-com/actions)
  [![Actions Status](https://github.com/2beens/serj-tubin-com/workflows/CodeQL/badge.svg)](https://github.com/2beens/serj-tubin-com/actions)
  [![Go Report Card](https://goreportcard.com/badge/github.com/2beens/serj-tubin-com)](https://goreportcard.com/report/github.com/2beens/serj-tubin-com)
  
  Personal tech sandbox and testing site 🕵️‍♀️
  
  Hosted and available at: https://www.serj-tubin.com/
  
  (also at https://2beens.xyz 🤷🏼‍)

</div>

* I use golang 🦫to make this all happen 👨🏼‍💻
* `Dockerfile` and `docker/` folder contain configs for running the service stack within Docker
* I use Aerospike to store those visitor board messages 💿
    * there are better choices than Aerospike, I know, just wanted to test Aerospike
    * update: I am thinking to replace it for something else
* I tried to use Elasticsearch Stack for logging and monitoring on a different server, but it was kinda hungry for memory, and exceeded the needs of this humble project (will try something else lighter)
* Used CloudFlare to put everything behind it, but only paid plans allowed passing of real client ip in the proxied requests
* ~~https://freegeoip.app~~ since 2022, ~~https://ipbase.com~~, dec 22: https://ipinfo.io
    * used for geo IP data
* http://api.openweathermap.org
    * used for weather data
* I use GitHub Workflow Actions for a part of CI/CD
    * unit testing
    * static code analysis
    * deploy on new release
* I use PostgreSQL to store blog posts and personal web history (netlog), notes, etc.
* I also use Redis to store session data
* Prometheus is used for metrics (then Grafana to visualize them)
* Honeycomb is used for distributed tracing
* Sentry for error tracking and alerting
* I use Vue to make the frontend part (was my first Vue project, so I don't like it)
    * source @ https://github.com/2beens/serj-tubin-vue
* ❗️ Disclaimer: some parts of the system are deliberately unoptimized or complicated for testing ☑️ / learning 👨🏼‍🏫 / trial 🛠 purposes

### TODO: Observability (done ✅)
- use otel to collect and send metrics and tracing data
  - for metrics use Prometheus
  - for tracing use free Honeycomb plan
    - https://ui.honeycomb.io/serj-tubin-com/environments/test/send-data#
