package glasses

import (
	"zjsj/internal/service"
)

type sGlasses struct{}

func init() {
	service.RegisterGlasses(New())
}

func New() service.IGlasses {
	return &sGlasses{}
}
