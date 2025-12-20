package di

import (
	"fmt"
	app "github.com/asp24/go-sf-di/examples/basic/app"
	"sync"
)

var param_dsn string = "postgres://localhost/app"
var param_log_prefix string = "[app] "

type Container struct {
	mu                    sync.Mutex
	svc_logger            *app.Logger
	svc_provider_paypal   *app.PaymentProvider
	svc_provider_stripe   *app.PaymentProvider
	svc_repo              *app.Repo
	svc_service           *app.Service
	svc_service_decorator *app.Service
}

func (c *Container) buildLogger() (*app.Logger, error) {
	return app.NewLogger(param_log_prefix), nil
}

func (c *Container) buildProviderPaypal() (*app.PaymentProvider, error) {
	return app.NewPaypalProvider(), nil
}

func (c *Container) buildProviderStripe() (*app.PaymentProvider, error) {
	return app.NewStripeProvider(), nil
}

func (c *Container) buildRepo() (*app.Repo, error) {
	return app.NewRepo(param_dsn), nil
}

func (c *Container) buildService() (*app.Service, error) {
	var zero *app.Service
	dep_repo, err := c.getRepo()
	if err != nil {
		return zero, fmt.Errorf("service %q arg[%d]: %w", "service", 0, err)
	}
	dep_logger, err := c.getLogger()
	if err != nil {
		return zero, fmt.Errorf("service %q arg[%d]: %w", "service", 1, err)
	}
	tag_provider_stripe, err := c.getProviderStripe()
	if err != nil {
		return zero, fmt.Errorf("service %q arg[%d] tag %q: %w", "service", 2, "payment.provider", err)
	}
	tag_provider_paypal, err := c.getProviderPaypal()
	if err != nil {
		return zero, fmt.Errorf("service %q arg[%d] tag %q: %w", "service", 2, "payment.provider", err)
	}
	res, err := app.NewService(dep_repo, dep_logger, []*app.PaymentProvider{tag_provider_stripe, tag_provider_paypal})
	if err != nil {
		return zero, fmt.Errorf("service %q constructor: %w", "service", err)
	}
	return res, nil
}

func (c *Container) buildServiceDecoratorDecorator(inner *app.Service) (*app.Service, error) {
	var zero *app.Service
	dep_logger, err := c.getLogger()
	if err != nil {
		return zero, fmt.Errorf("service %q arg[%d]: %w", "service.decorator", 1, err)
	}
	return app.DecorateService(inner, dep_logger), nil
}

func (c *Container) buildDecoratedServiceDecorator() (*app.Service, error) {
	var zero *app.Service
	inner, err := c.buildService()
	if err != nil {
		return zero, fmt.Errorf("service %q base %q: %w", "service.decorator", "service", err)
	}
	inner, err = c.buildServiceDecoratorDecorator(inner)
	if err != nil {
		return zero, fmt.Errorf("service %q decorator %q: %w", "service.decorator", "service.decorator", err)
	}
	return inner, nil
}

func (c *Container) getLogger() (*app.Logger, error) {
	var zero *app.Logger
	if c.svc_logger != nil {
		return c.svc_logger, nil
	}
	res, err := c.buildLogger()
	if err != nil {
		return zero, err
	}
	c.svc_logger = res
	return res, nil
}

func (c *Container) getProviderPaypal() (*app.PaymentProvider, error) {
	var zero *app.PaymentProvider
	if c.svc_provider_paypal != nil {
		return c.svc_provider_paypal, nil
	}
	res, err := c.buildProviderPaypal()
	if err != nil {
		return zero, err
	}
	c.svc_provider_paypal = res
	return res, nil
}

func (c *Container) getProviderStripe() (*app.PaymentProvider, error) {
	var zero *app.PaymentProvider
	if c.svc_provider_stripe != nil {
		return c.svc_provider_stripe, nil
	}
	res, err := c.buildProviderStripe()
	if err != nil {
		return zero, err
	}
	c.svc_provider_stripe = res
	return res, nil
}

func (c *Container) getRepo() (*app.Repo, error) {
	var zero *app.Repo
	if c.svc_repo != nil {
		return c.svc_repo, nil
	}
	res, err := c.buildRepo()
	if err != nil {
		return zero, err
	}
	c.svc_repo = res
	return res, nil
}

func (c *Container) getService() (*app.Service, error) {
	var zero *app.Service
	if c.svc_service != nil {
		return c.svc_service, nil
	}
	res, err := c.buildDecoratedServiceDecorator()
	if err != nil {
		return zero, err
	}
	c.svc_service = res
	return res, nil
}

func (c *Container) getServiceDecorator() (*app.Service, error) {
	var zero *app.Service
	if c.svc_service_decorator != nil {
		return c.svc_service_decorator, nil
	}
	res, err := c.buildDecoratedServiceDecorator()
	if err != nil {
		return zero, err
	}
	c.svc_service_decorator = res
	return res, nil
}

func (c *Container) GetService() (*app.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getService()
}
