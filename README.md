# GORM Oracle Driver

## Description

GORM Oracle driver for connect Oracle DB and Manage Oracle DB, Based on [CengSin/oracle](https://github.com/CengSin/oracle)
and [sijms/go-ora](https://github.com/sijms/go-ora) (pure go oracle client)，*not recommended for use in a production environment*.

## Required dependency Install

- Oracle `11g` + (*`v1.6.3` and earlier versions support only `12c` +*)
- Golang
  - `v1.6.1`: `go1.16` +
  - `v1.6.2`: `go1.18` +
- gorm `1.24.0` +

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
		DSN:                     url,
		IgnoreCase:              false, // query conditions are not case-sensitive
		NamingCaseSensitive:     true,  // whether naming is case-sensitive
		VarcharSizeIsCharLength: true,  // whether VARCHAR type size is character length, defaulting to byte length

		// RowNumberAliasForOracle11 is the alias for ROW_NUMBER() in Oracle 11g, defaulting to ROW_NUM
		RowNumberAliasForOracle11: "ROW_NUM",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction:                   true, // 是否禁用默认在事务中执行单次创建、更新、删除操作
		DisableForeignKeyConstraintWhenMigrating: true, // 是否禁止在自动迁移或创建表时自动创建外键约束
		// 自定义命名策略
		NamingStrategy: schema.NamingStrategy{
			NoLowerCase:         true, // 是否不自动转换小写表名
			IdentifierMaxLength: 30,   // Oracle: 30, PostgreSQL:63, MySQL: 64, SQL Server、SQLite、DM: 128
		},
		PrepareStmt:     false, // 创建并缓存预编译语句，启用后可能会报 ORA-01002 错误
		CreateBatchSize: 50,    // 插入数据默认批处理大小
	})
	if err != nil {
		// panic error or log error info
	}

	// set session parameters
	if sqlDB, err := db.DB(); err == nil {
		_, _ = oracle.AddSessionParams(sqlDB, map[string]string{
			"TIME_ZONE":               "+08:00",                       // ALTER SESSION SET TIME_ZONE = '+08:00';
			"NLS_DATE_FORMAT":         "YYYY-MM-DD",                   // ALTER SESSION SET NLS_DATE_FORMAT = 'YYYY-MM-DD';
			"NLS_TIME_FORMAT":         "HH24:MI:SSXFF",                // ALTER SESSION SET NLS_TIME_FORMAT = 'HH24:MI:SS.FF3';
			"NLS_TIMESTAMP_FORMAT":    "YYYY-MM-DD HH24:MI:SSXFF",     // ALTER SESSION SET NLS_TIMESTAMP_FORMAT = 'YYYY-MM-DD HH24:MI:SS.FF3';
			"NLS_TIME_TZ_FORMAT":      "HH24:MI:SS.FF TZR",            // ALTER SESSION SET NLS_TIME_TZ_FORMAT = 'HH24:MI:SS.FF3 TZR';
			"NLS_TIMESTAMP_TZ_FORMAT": "YYYY-MM-DD HH24:MI:SSXFF TZR", // ALTER SESSION SET NLS_TIMESTAMP_TZ_FORMAT = 'YYYY-MM-DD HH24:MI:SS.FF3 TZR';
		})
	}

	// do somethings
}

```

## Questions

<!--suppress HtmlDeprecatedAttribute -->
<details>
<summary>ORA-01000: 超出打开游标的最大数</summary>

> ORA-00604: 递归 SQL 级别 1 出现错误
> 
> ORA-01000: 超出打开游标的最大数

```shell
show parameter OPEN_CURSORS;
```

```sql
alter system set OPEN_CURSORS = 1000; -- or bigger
commit;
```

</details>

<details>
<summary>ORA-01002: 提取违反顺序</summary>

> 如果重复执行同一查询，第一次查询成功，第二次报 `ORA-01002` 错误，可能是因为启用了 `PrepareStmt`，关闭此配置即可。

推荐配置：

```go
&gorm.Config{
    SkipDefaultTransaction:                   true, // 是否禁用默认在事务中执行单次创建、更新、删除操作
    DisableForeignKeyConstraintWhenMigrating: true, // 是否禁止在自动迁移或创建表时自动创建外键约束
    // 自定义命名策略
    NamingStrategy: schema.NamingStrategy{
        NoLowerCase:         true, // 是否不自动转换小写表名
        IdentifierMaxLength: 30,   // Oracle: 30, PostgreSQL:63, MySQL: 64, SQL Server、SQLite、DM: 128
    },
    PrepareStmt:     false, // 创建并缓存预编译语句，启用后可能会报 ORA-01002 错误
    CreateBatchSize: 50,    // 插入数据默认批处理大小
}
```

</details>

## Contributors

<!-- readme: collaborators,contributors -start -->
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
        <a href="https://github.com/cloorc">
            <img src="https://avatars.githubusercontent.com/u/13597105?v=4" width="100;" alt="cloorc"/>
            <br />
            <sub><b>cloorc</b></sub>
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
    </td></tr>
<tr>
    <td align="center">
        <a href="https://github.com/cengsin">
            <img src="https://avatars.githubusercontent.com/u/23357893?v=4" width="100;" alt="cengsin"/>
            <br />
            <sub><b>cengsin</b></sub>
        </a>
    </td></tr>
</table>
<!-- readme: collaborators,contributors -end -->

## LICENSE

[MIT license](./LICENSE)

- Copyright (c) 2020 [Jinzhu](https://github.com/jinzhu)
- Copyright (c) 2020 [Steve Fan](https://github.com/stevefan1999-personal)
- Copyright (c) 2020 [CengSin](https://github.com/CengSin)
- Copyright (c) 2022 [dzwvip](https://github.com/dzwvip)
- Copyright (c) 2022-present [iTanken](https://github.com/iTanken)
