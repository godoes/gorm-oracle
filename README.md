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

```go
package main

import (
	oracle "github.com/godoes/gorm-oracle"
	goora "github.com/sijms/go-ora/v2"
	"gorm.io/gorm"
)

func main() {
	options := map[string]string{
		"CONNECTION TIMEOUT": "90",
		"LANGUAGE":           "SIMPLIFIED CHINESE",
		"TERRITORY":          "CHINA",
		"SSL":                "false",
	}
	// oracle://user:password@127.0.0.1:1521/service
	url := oracle.BuildUrl("127.0.0.1", "1521", "service", "user", "password", options)
	dialector := oracle.New(oracle.Config{
		DSN:                 url,
		IgnoreCase:          false, // query conditions are not case-sensitive
		NamingCaseSensitive: true,  // whether naming is case-sensitive
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		// panic error or log error info
	}

	// set session parameters
	if sqlDB, err := db.DB(); err == nil {
		_ = goora.AddSessionParam(sqlDB, "TIME_ZONE", "+08:00")                                     // ALTER SESSION SET TIME_ZONE = '+08:00';
		_ = goora.AddSessionParam(sqlDB, "NLS_DATE_FORMAT", "YYYY-MM-DD")                           // ALTER SESSION SET NLS_DATE_FORMAT = 'YYYY-MM-DD';
		_ = goora.AddSessionParam(sqlDB, "NLS_TIME_FORMAT", "HH24:MI:SSXFF")                        // ALTER SESSION SET NLS_TIME_FORMAT = 'HH24:MI:SS.FF3';
		_ = goora.AddSessionParam(sqlDB, "NLS_TIMESTAMP_FORMAT", "YYYY-MM-DD HH24:MI:SSXFF")        // ALTER SESSION SET NLS_TIMESTAMP_FORMAT = 'YYYY-MM-DD HH24:MI:SS.FF3';
		_ = goora.AddSessionParam(sqlDB, "NLS_TIME_TZ_FORMAT", "HH24:MI:SS.FF TZR")                 // ALTER SESSION SET NLS_TIME_TZ_FORMAT = 'HH24:MI:SS.FF3 TZR';
		_ = goora.AddSessionParam(sqlDB, "NLS_TIMESTAMP_TZ_FORMAT", "YYYY-MM-DD HH24:MI:SSXFF TZR") // ALTER SESSION SET NLS_TIMESTAMP_TZ_FORMAT = 'YYYY-MM-DD HH24:MI:SS.FF3 TZR';
	}

	// do somethings
}

```

## Contributors

<!--suppress HtmlDeprecatedAttribute -->
<!-- readme: collaborators,dzwvip,jinzhu,miclle,stevefan1999-personal,zhangzetao,CengSin/- -start -->
<table>
<tr>
    <td align="center">
        <a href="https://github.com/iTanken">
            <img src="https://avatars.githubusercontent.com/u/23544702?v=4" width="100;" alt="iTanken"/>
            <br />
            <sub><b>iTanken</b></sub>
        </a>
    </td>
    <td align="center">
        <a href="https://github.com/dzwvip">
            <img src="https://avatars.githubusercontent.com/u/17947637?v=4" width="100;" alt="dzwvip"/>
            <br />
            <sub><b>dzwvip</b></sub>
        </a>
    </td>
    <td align="center">
        <a href="https://github.com/jinzhu">
            <img src="https://avatars.githubusercontent.com/u/6843?v=4" width="100;" alt="jinzhu"/>
            <br />
            <sub><b>jinzhu</b></sub>
        </a>
    </td>
    <td align="center">
        <a href="https://github.com/miclle">
            <img src="https://avatars.githubusercontent.com/u/186694?v=4" width="100;" alt="miclle"/>
            <br />
            <sub><b>miclle</b></sub>
        </a>
    </td>
    <td align="center">
        <a href="https://github.com/stevefan1999-personal">
            <img src="https://avatars.githubusercontent.com/u/29133953?v=4" width="100;" alt="stevefan1999-personal"/>
            <br />
            <sub><b>stevefan1999-personal</b></sub>
        </a>
    </td>
    <td align="center">
        <a href="https://github.com/zhangzetao">
            <img src="https://avatars.githubusercontent.com/u/15045771?v=4" width="100;" alt="zhangzetao"/>
            <br />
            <sub><b>zhangzetao</b></sub>
        </a>
    </td></tr>
</table>
<!-- readme: collaborators,dzwvip,jinzhu,miclle,stevefan1999-personal,zhangzetao,CengSin/- -end -->

## LICENSE

[MIT license](./LICENSE)

- Copyright (c) 2022-present [iTanken](https://github.com/iTanken)
- Copyright (c) 2022 [dzwvip](https://github.com/dzwvip)
- Copyright (c) 2020 [CengSin](https://github.com/CengSin)
- Copyright (c) 2020 [Steve Fan](https://github.com/stevefan1999-personal)
- Copyright (c) 2020 [Jinzhu](https://github.com/jinzhu)
