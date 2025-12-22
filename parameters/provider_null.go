package parameters

type ProviderNull struct {
}

var ProviderNullInstance = &ProviderNull{}

func NewProviderNull() *ProviderNull {
	return ProviderNullInstance
}

func (p *ProviderNull) Has(_ string) bool {
	return false
}

func (p *ProviderNull) GetString(_ string) (string, error) {
	return "", ErrParameterNotFound
}

func (p *ProviderNull) GetInt(_ string) (int, error) {
	return 0, ErrParameterNotFound
}

func (p *ProviderNull) GetBool(_ string) (bool, error) {
	return false, ErrParameterNotFound
}

func (p *ProviderNull) GetFloat(_ string) (float64, error) {
	return 0.0, ErrParameterNotFound
}
