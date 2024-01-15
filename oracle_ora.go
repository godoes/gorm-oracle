package oracle

import "github.com/sijms/go-ora/v2"

type (
	RefCursor struct {
		go_ora.RefCursor
	}

	DataSet struct {
		go_ora.DataSet
	}

	Out struct {
		go_ora.Out
	}
)

func (cursor *RefCursor) Query() (dataset *DataSet, err error) {
	var d *go_ora.DataSet
	if d, err = cursor.RefCursor.Query(); err != nil {
		return
	}
	dataset = &DataSet{DataSet: *d}
	return
}
