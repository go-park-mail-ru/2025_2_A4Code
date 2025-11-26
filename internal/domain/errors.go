package domain

import "errors"

var ErrFolderExists = errors.New("folder already exists")
var ErrFolderNotFound = errors.New("folder not found")
var ErrFolderSystem = errors.New("cannot delete system folder")
