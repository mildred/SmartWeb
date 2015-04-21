package server

type AuthMethod struct {
	Name   string
	Params map[string]string
}

type HeaderParse struct {
	Methods []AuthMethod
}

func (p *HeaderParse) addAuthScheme(method string) {
	p.Methods = append(p.Methods, AuthMethod{
		Name:   method,
		Params: make(map[string]string),
	})
}

func (p *HeaderParse) setB64Param(param string) {
	i := len(p.Methods) - 1
	p.Methods[i].Params["base64"] = param
}

func (p *HeaderParse) setParam(key, value string) {
	i := len(p.Methods) - 1
	p.Methods[i].Params[key] = value
}
