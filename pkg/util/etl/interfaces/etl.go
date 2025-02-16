package etl

type ETL interface {
	Extract() (interface{}, error)
	Transform(data interface{}) (interface{}, error)
	Load(data interface{}) error
}
