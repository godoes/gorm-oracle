package clauses

import (
	"gorm.io/gorm/clause"
)

type WhenNotMatched struct {
	clause.Values
	Where clause.Where
}

func (w WhenNotMatched) Name() string {
	return "WHEN NOT MATCHED"
}

func (w WhenNotMatched) Build(builder clause.Builder) {
	if len(w.Columns) > 0 {
		if len(w.Values.Values) != 1 {
			panic("cannot insert more than one rows due to Oracle SQL language restriction")
		}

		_, _ = builder.WriteString(" THEN")
		_, _ = builder.WriteString(" INSERT ")
		w.Build(builder)

		if len(w.Where.Exprs) > 0 {
			_, _ = builder.WriteString(w.Where.Name())
			_ = builder.WriteByte(' ')
			w.Where.Build(builder)
		}
	}
}
