# GORM Oracle Driver

## Description

GORM Oracle driver for connect Oracle DB and Manage Oracle DB, Based on [CengSin/oracle](https://github.com/CengSin/oracle)
and [sijms/go-ora](https://github.com/sijms/go-ora) (pure go oracle client)，not recommended for use in a production environment.

## Required dependency Install

- Oracle 12C+
- Golang 1.16+
- gorm 1.24.0+

## Quick Start

### How to install 

```bash
go get -d github.com/godoes/gorm-oracle
```

### Usage

```shell
package main

import (
	oracle "github.com/godoes/gorm-oracle"
	"gorm.io/gorm"
)

func main() {
	// oracle://user:password@127.0.0.1:1521/service
	url := oracle.BuildUrl("127.0.0.1", "1521", "service", "user", "password", nil)
	db, err := gorm.Open(oracle.Open(url), &gorm.Config{})
	if err != nil {
		// panic error or log error info
	}

	// do somethings
}
```
