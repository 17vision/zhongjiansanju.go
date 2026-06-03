package glasses_use

import "zjsj/internal/service"

type sGlassesUse struct{}

func init() {
	service.RegisterGlassesUse(New())
}

func New() service.IGlassesUse {
	return &sGlassesUse{}
}
