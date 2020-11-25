
## Title

[![codecov](https://codecov.io/gh/avayau/rorm/branch/master/graph/badge.svg)](https://codecov.io/gh/avayau/rorm)
Rorm is a tiny orm system for redis based on go-redis V2.It supports save golang struct to redis HMAP,and also support load data from redis to struct.It works like gorm.If you implement interface RormLoader,Rorm will automatically load data if data not exists in redis.

## Table of Contents

- [Title](#title)
- [Table of Contents](#table-of-contents)
- [Install](#install)
- [Usage](#usage)
- [TODO](#todo)
- [License](#license)

## Install

To install Rorm package, you need to install Go and set your Go workspace first.
Rorm only test on golang version 1.15+

```sh
$ go get -u github.com/avayayu/rorm
```


## Usage

- 
```
```

## TODO 


- cluster support
- callback support
- more redis driver support


## License

MIT © Richard McRichface