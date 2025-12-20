package di

import (
	"fmt"
	app "github.com/asp24/go-sf-di/examples/advanced/app"
	"sync"
)

var param_dsn string = "postgres://localhost/advanced"
var param_log_prefix string = "[us-east-1] "
var param_mail_host string = "smtp.us.example"
var param_mail_prefix string = "ADV-"
var param_mail_retries int = 5

type Container struct {
	mu                sync.Mutex
	svc_db            *app.DB
	svc_factory       *app.Factory
	svc_logger        *app.Logger
	svc_mailer        *app.Mailer
	svc_mailer_prefix *app.Mailer
	svc_mailer_retry  *app.Mailer
}

func (c *Container) buildDb() (*app.DB, error) {
	var zero *app.DB
	res, err := app.NewDB(param_dsn)
	if err != nil {
		return zero, fmt.Errorf("service %q constructor: %w", "db", err)
	}
	return res, nil
}

func (c *Container) buildFactory() (*app.Factory, error) {
	var zero *app.Factory
	dep_logger, err := c.getLogger()
	if err != nil {
		return zero, fmt.Errorf("service %q arg[%d]: %w", "factory", 0, err)
	}
	return app.NewFactory(dep_logger), nil
}

func (c *Container) buildHandler() (*app.Handler, error) {
	var zero *app.Handler
	dep_db, err := c.getDb()
	if err != nil {
		return zero, fmt.Errorf("service %q arg[%d]: %w", "handler", 0, err)
	}
	dep_mailer, err := c.getMailer()
	if err != nil {
		return zero, fmt.Errorf("service %q arg[%d]: %w", "handler", 1, err)
	}
	recv_handler, err := c.getFactory()
	if err != nil {
		return zero, fmt.Errorf("service %q receiver %q: %w", "handler", "factory", err)
	}
	return recv_handler.NewHandler(dep_db, dep_mailer), nil
}

func (c *Container) buildLogger() (*app.Logger, error) {
	return app.NewLogger(param_log_prefix), nil
}

func (c *Container) buildMailer() (*app.Mailer, error) {
	return app.NewMailer(param_mail_host), nil
}

func (c *Container) buildMailerPrefixDecorator(inner *app.Mailer) (*app.Mailer, error) {
	return app.AddPrefix(inner, param_mail_prefix), nil
}

func (c *Container) buildMailerRetryDecorator(inner *app.Mailer) (*app.Mailer, error) {
	return app.AddRetry(inner, param_mail_retries), nil
}

func (c *Container) buildNotifierEmail() (*app.EmailNotifier, error) {
	var zero *app.EmailNotifier
	dep_mailer, err := c.getMailer()
	if err != nil {
		return zero, fmt.Errorf("service %q arg[%d]: %w", "notifier.email", 0, err)
	}
	return app.NewEmailNotifier(dep_mailer), nil
}

func (c *Container) buildNotifierSms() (app.SMSNotifier, error) {
	return app.NewSMSNotifier(), nil
}

func (c *Container) buildDecoratedMailerPrefix() (*app.Mailer, error) {
	var zero *app.Mailer
	inner, err := c.buildMailer()
	if err != nil {
		return zero, fmt.Errorf("service %q base %q: %w", "mailer.prefix", "mailer", err)
	}
	inner, err = c.buildMailerRetryDecorator(inner)
	if err != nil {
		return zero, fmt.Errorf("service %q decorator %q: %w", "mailer.prefix", "mailer.retry", err)
	}
	inner, err = c.buildMailerPrefixDecorator(inner)
	if err != nil {
		return zero, fmt.Errorf("service %q decorator %q: %w", "mailer.prefix", "mailer.prefix", err)
	}
	return inner, nil
}

func (c *Container) buildDecoratedMailerRetry() (*app.Mailer, error) {
	var zero *app.Mailer
	inner, err := c.buildMailer()
	if err != nil {
		return zero, fmt.Errorf("service %q base %q: %w", "mailer.retry", "mailer", err)
	}
	inner, err = c.buildMailerRetryDecorator(inner)
	if err != nil {
		return zero, fmt.Errorf("service %q decorator %q: %w", "mailer.retry", "mailer.retry", err)
	}
	return inner, nil
}

func (c *Container) getDb() (*app.DB, error) {
	var zero *app.DB
	if c.svc_db != nil {
		return c.svc_db, nil
	}
	res, err := c.buildDb()
	if err != nil {
		return zero, err
	}
	c.svc_db = res
	return res, nil
}

func (c *Container) getFactory() (*app.Factory, error) {
	var zero *app.Factory
	if c.svc_factory != nil {
		return c.svc_factory, nil
	}
	res, err := c.buildFactory()
	if err != nil {
		return zero, err
	}
	c.svc_factory = res
	return res, nil
}

func (c *Container) getHandler() (*app.Handler, error) {
	var zero *app.Handler
	res, err := c.buildHandler()
	if err != nil {
		return zero, err
	}
	return res, nil
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

func (c *Container) getMailer() (*app.Mailer, error) {
	var zero *app.Mailer
	if c.svc_mailer != nil {
		return c.svc_mailer, nil
	}
	res, err := c.buildDecoratedMailerPrefix()
	if err != nil {
		return zero, err
	}
	c.svc_mailer = res
	return res, nil
}

func (c *Container) getMailerPrefix() (*app.Mailer, error) {
	var zero *app.Mailer
	if c.svc_mailer_prefix != nil {
		return c.svc_mailer_prefix, nil
	}
	res, err := c.buildDecoratedMailerPrefix()
	if err != nil {
		return zero, err
	}
	c.svc_mailer_prefix = res
	return res, nil
}

func (c *Container) getMailerRetry() (*app.Mailer, error) {
	var zero *app.Mailer
	if c.svc_mailer_retry != nil {
		return c.svc_mailer_retry, nil
	}
	res, err := c.buildDecoratedMailerRetry()
	if err != nil {
		return zero, err
	}
	c.svc_mailer_retry = res
	return res, nil
}

func (c *Container) getNotifierEmail() (*app.EmailNotifier, error) {
	var zero *app.EmailNotifier
	res, err := c.buildNotifierEmail()
	if err != nil {
		return zero, err
	}
	return res, nil
}

func (c *Container) getNotifierSms() (app.SMSNotifier, error) {
	var zero app.SMSNotifier
	res, err := c.buildNotifierSms()
	if err != nil {
		return zero, err
	}
	return res, nil
}

func (c *Container) GetDb() (*app.DB, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getDb()
}

func (c *Container) GetFactory() (*app.Factory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getFactory()
}

func (c *Container) GetHandler() (*app.Handler, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getHandler()
}

func (c *Container) GetLogger() (*app.Logger, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getLogger()
}

func (c *Container) GetMailer() (*app.Mailer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getMailer()
}

func (c *Container) GetMailerPrefix() (*app.Mailer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getMailerPrefix()
}

func (c *Container) GetMailerRetry() (*app.Mailer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getMailerRetry()
}

func (c *Container) GetNotifierEmail() (*app.EmailNotifier, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getNotifierEmail()
}

func (c *Container) GetNotifierSms() (app.SMSNotifier, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getNotifierSms()
}
