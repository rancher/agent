package model

type ContainerNotFoundError struct {
}

func (c ContainerNotFoundError) Error() string {
	return "Container not found"
}
