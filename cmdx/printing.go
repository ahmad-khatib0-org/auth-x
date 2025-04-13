package cmdx

type (
	TableHeader interface {
		Header() []string
	}
	TableRow interface {
		TableHeader
		Columns() []string
		Interface() any
	}
	Table interface {
		TableHeader
		Table() [][]string
		Interface() any
		Len() int
	}
	Nil    struct{}
	format string
)
