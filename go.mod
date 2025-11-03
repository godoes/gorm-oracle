module github.com/godoes/gorm-oracle

go 1.18

require (
	github.com/emirpasic/gods v1.18.1
	github.com/sijms/go-ora/v2 v2.9.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.20.0 // indirect
)

exclude (
	github.com/sijms/go-ora/v2 v2.8.8 // ORA-03137: 来自客户机的格式错误的 TTC 包被拒绝: [opiexe: protocol violation]
	github.com/sijms/go-ora/v2 v2.8.9 // has bug
)

retract (
	v1.5.12
	v1.5.1
	v1.5.0
)
