package clauses

import (
	"gorm.io/gorm/clause"
)

type WhenMatched struct {
	clause.Set
	Where, Delete clause.Where
}

func (w WhenMatched) Name() string {
	return "WHEN MATCHED"
}

func (w WhenMatched) Build(builder clause.Builder) {
	if len(w.Set) > 0 {
		_, _ = builder.WriteString(" THEN")
		_, _ = builder.WriteString(" UPDATE ")
		_, _ = builder.WriteString(w.Name())
		_ = builder.WriteByte(' ')
		w.Build(builder)

		buildWhere := func(where clause.Where) {
			_, _ = builder.WriteString(where.Name())
			_ = builder.WriteByte(' ')
			where.Build(builder)
		}

		if len(w.Where.Exprs) > 0 {
			buildWhere(w.Where)
		}

		if len(w.Delete.Exprs) > 0 {
			_, _ = builder.WriteString(" DELETE ")
			buildWhere(w.Delete)
		}
	}
}
