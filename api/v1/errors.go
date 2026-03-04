package v1

var (
	// ErrSuccess common errors
	ErrSuccess             = newError(0, "ok")
	ErrBadRequest          = newError(400, "Bad Request")
	ErrUnauthorized        = newError(401, "未授权的API Key")
	ErrNotFound            = newError(404, "Not Found")
	ErrInternalServerError = newError(500, "Internal Server Error")

	// ErrEmailAlreadyUse more biz errors
	ErrEmailAlreadyUse = newError(1001, "The email is already in use.")
	ErrUnInputAPIkey   = newError(1002, "请输入api key")
	ErrExistConnect    = newError(1003, "已存在连接")
)
