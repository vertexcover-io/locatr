package main

import "errors"

var (
	FailedToCreateLlmClient   = errors.New("failed to create LLM client")
	ErrClientNotInstantiated  = errors.New("client not instantiated")
	ErrFailedToRetrieveLocatr = errors.New("failed to retrieve locatr")
	ErrCdpConnectionCreation  = errors.New("error while creating CDP connection")
	ErrCdpLocatrCreation      = errors.New("error while creating CDP locatr")
	ErrSeleniumLocatrCreation = errors.New("error while creating Selenium locatr")
)
