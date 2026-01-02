package app

import (
	"context"
)

type Job interface {
	Run(ctx context.Context) error
}

type PaymentProvider interface {
	Pay(sum int) error
}

type PaymentProviderDummy struct{}

func NewPaymentProviderDummy() *PaymentProviderDummy {
	return &PaymentProviderDummy{}
}

func (p *PaymentProviderDummy) Pay(sum int) error {
	return nil
}

type PaymentProviderCommissionDecorator struct {
	inner      PaymentProvider
	commission int
}

func NewPaymentProviderCommissionDecorator(inner PaymentProvider, commission int) *PaymentProviderCommissionDecorator {
	return &PaymentProviderCommissionDecorator{inner: inner, commission: commission}
}

func (p *PaymentProviderCommissionDecorator) Pay(sum int) error {
	return p.inner.Pay(sum + p.commission)
}

func (p *PaymentProviderCommissionDecorator) Run(ctx context.Context) error {
	<-ctx.Done()

	return nil
}
