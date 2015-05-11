package deb

type Account uint16
type Date uint32
type Moment uint64
type DateRange struct{ Start, End Date }
type MomentRange struct{ Start, End Moment }
type Array [][][]int64
type Entries map[Account]int64
type Transaction struct {
	Moment  Moment
	Date    Date
	Entries Entries
}

type Space interface {
	Append(s Space)
	Slice(a []Account, d []DateRange, m []MomentRange) Space
	Projection(a []Account, d []DateRange, m []MomentRange) Space
	Transactions() chan *Transaction
}
