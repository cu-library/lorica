# lorica
[![Build Status](https://travis-ci.org/cu-library/lorica.svg)](https://travis-ci.org/cu-library/lorica)
[![Go Report Card](https://goreportcard.com/badge/github.com/cu-library/lorica)](https://goreportcard.com/report/github.com/cu-library/lorica)

*An authenticating proxy for the Summon API.*

The Summon API has two problems which make it difficult to use with client-side javascript applications: 

* It does not support CORS. 
* It requires authentication. 

Similar to work done by [Jonathan Rochkind](https://bibwild.wordpress.com/2013/06/20/an-aid-for-those-developing-against-the-summon-api/) and others, this proxy addresses those issues. 

* It supports CORS requests, including properly handling preflight OPTIONS requests. 
* It authorizes requests to the API on behalf of the client. 

Requests to Lorica will be authorized and sent to the Summon API. For example, if Lorica is running on localhost at port 8877, accessing:

`http://localhost:8877/2.0.0/search/ping`

will return the response from 

`http://api.summon.serialssolutions.com/2.0.0/search/ping`

By default, Lorica runs with a rate limiter, to disuade malicious users from scraping the Summon API using the provided credentials.

Lorica is designed with http://12factor.net/ in mind. 

```
Lorica: An authenticating proxy for the Summon API

  -accessid string
        Access ID
  -address string
        Address for the server to bind on. (default ":8877")
  -allowedorigins string
        A list of allowed origins for CORS, delimited by the ; character. To allow any origin to connect, use *.
  -checkproxyheaders
        Have the rate limiter use the IP address from the X-Forwarded-For and X-Real-IP header first. You may need this if you are running Lorica behind a proxy.
  -loglevel string
        The maximum log level which will be logged. error < warn < info < debug < trace. For example, trace will log everything, info will log info, warn, and error. (default "warn")
  -maxrequests int
        The maximum number of requests accepted from one client per one second interval. (default 1)
  -ratelimit
        Enable and disable rate limiting. (default true)
  -secretkey string
        Secret Key
  -summonapi string
        Summon API URL. (default "http://api.summon.serialssolutions.com")
  -timeout int
        The number of seconds to wait for a response from Summon. (default 10)
  The possible environment variables:
  LORICA_ACCESSID
  LORICA_ADDRESS
  LORICA_ALLOWEDORIGINS
  LORICA_CHECKPROXYHEADERS
  LORICA_LOGLEVEL
  LORICA_MAXREQUESTS
  LORICA_RATELIMIT
  LORICA_SECRETKEY
  LORICA_SUMMONAPI
  LORICA_TIMEOUT
```
