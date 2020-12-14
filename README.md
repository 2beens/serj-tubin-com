# serj-tubin.com
[![Actions Status](https://github.com/2beens/serj-tubin-com/workflows/CI/badge.svg)](https://github.com/2beens/serj-tubin-com/actions)
[![Actions Status](https://github.com/2beens/serj-tubin-com/workflows/CodeQL/badge.svg)](https://github.com/2beens/serj-tubin-com/actions)

Personal garbage ♻️ and testing site 🕵️‍♀️

Hosted and available at: https://www.serj-tubin.com/

* I use golang to make this all happen 👨🏼‍💻
* I use Aerospike to store those visitor board messages 💿
    * there are better choices than Aerospike, I know, just wanted to test Aerospike
* I tried to use Elasticsearch Stack for logging and monitoring on a different server, but it was kinda hungry for memory, and exceeded the needs of this humble project (will try something else lighter)
* https://freegeoip.app/
    * used for geo IP data
* http://api.openweathermap.org
    * used for weather data
* I use GitHub Workflow Actions for a part of CI
    * unit testing
    * static code analysis
* I use PostgreSQL to store blog posts
* I use Vue to make the frontend part
    * source @ https://github.com/2beens/serj-tubin-vue
