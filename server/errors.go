package main

import "errors"

var (
	// locatr creation errors
	FailedToCreateLlmClient   = errors.New("failed to create LLM client")
	ErrClientNotInstantiated  = errors.New("client not instantiated")
	ErrFailedToRetrieveLocatr = errors.New("failed to retrieve locatr")
	ErrCdpConnectionCreation  = errors.New("error while creating CDP connection")
	ErrCdpLocatrCreation      = errors.New("error while creating CDP locatr")
	ErrSeleniumLocatrCreation = errors.New("error while creating Selenium locatr")

	// validation errors
	ErrInputMessageValidationFailed = errors.New("input message validation failed")
	ErrMissingLocatrSettings        = errors.New("missing locatrSettings for 'initial_handshake' type")
	ErrMissingSeleniumUrl           = errors.New("selenium plugin type selected but 'selenium_url' is missing or empty")
	ErrMissingSeleniumSessionId     = errors.New("selenium plugin type selected but 'selenium_session_id' is missing or empty")
	ErrMissingCdpUrl                = errors.New("cdp plugin type selected but 'cdp_url' is missing or empty")
	ErrInvalidPluginType            = errors.New("invalid 'plugin_type' provided")
	ErrEmptyUserRequest             = errors.New("empty 'Input' field provided")
)
