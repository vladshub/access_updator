### Access Updator

![Build](https://github.com/vladshub/access_updator/workflows/Go/badge.svg)

This project was ment to allow updating ingress rule update for projects where the access is IP based

## Usage:
```
Usage of access_updator:
  -group string
    	the AWS group id
  -ip string
    	the IP address we want to add
  -user string
    	the name of the user we want to give access to
```

Example: `access_updator -ip 212.132.54.21 -user vladshub -group sg-xxxxxx`
