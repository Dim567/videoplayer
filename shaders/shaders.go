package shaders

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Shader uint32

func New(
	vertexPath,
	fragmetPath string,
) uint32 {
	var vertexCode, fragmentCode string

	vShaderFileBuf, err := os.ReadFile(vertexPath)
	if err != nil {
		panic(err)
	}

	fShaderFileBuf, err := os.ReadFile(fragmetPath)
	if err != nil {
		panic(err)
	}

	vertexCode = string(vShaderFileBuf) + "\x00"
	fragmentCode = string(fShaderFileBuf) + "\x00"

	var vertex, fragment uint32
	vertex = gl.CreateShader(gl.VERTEX_SHADER)
	vShaderCode, free := gl.Strs(vertexCode)
	gl.ShaderSource(vertex, 1, vShaderCode, nil)
	free()
	gl.CompileShader(vertex)
	err = checkCompileErrors(vertex, "VERTEX")
	if err != nil {
		panic(err)
	}

	fragment = gl.CreateShader(gl.FRAGMENT_SHADER)

	fShaderCode, free := gl.Strs(fragmentCode)
	gl.ShaderSource(fragment, 1, fShaderCode, nil)
	free()
	gl.CompileShader(fragment)
	err = checkCompileErrors(fragment, "FRAGMENT")
	if err != nil {
		panic(err)
	}

	shaderID := gl.CreateProgram()
	gl.AttachShader(shaderID, vertex)
	gl.AttachShader(shaderID, fragment)
	gl.LinkProgram(shaderID)
	err = checkCompileErrors(shaderID, "PROGRAM")
	if err != nil {
		panic(err)
	}
	return shaderID
}

func Use(shaderId uint32) {
	gl.UseProgram(shaderId)
}

func SetBool(shID uint32, name string, value bool) {
	convertedValue := int32(0)
	if value {
		convertedValue = 1
	}
	gl.Uniform1i(gl.GetUniformLocation(shID, gl.Str(name+"\x00")), convertedValue)
}

func SetInt(shID uint32, name string, value int32) {
	gl.Uniform1i(gl.GetUniformLocation(shID, gl.Str(name+"\x00")), value)
}

func SetFloat(shID uint32, name string, value float32) {
	gl.Uniform1f(gl.GetUniformLocation(shID, gl.Str(name+"\x00")), value)
}

func SetVec4(shID uint32, name string, value *mgl32.Vec4) {
	gl.Uniform4fv(gl.GetUniformLocation(shID, gl.Str(name+"\x00")), 1, &value[0])
}

func SetMat4(shID uint32, name string, matrix *mgl32.Mat4) {
	locationName := gl.Str(name + "\x00")
	matrixId := gl.GetUniformLocation(shID, locationName)
	gl.UniformMatrix4fv(matrixId, 1, false, &matrix[0])
}

func checkCompileErrors(shader uint32, shaderType string) error {
	var success int32
	infoLog := strings.Repeat("\x00", 1024)

	if shaderType != "PROGRAM" {
		gl.GetShaderiv(shader, gl.COMPILE_STATUS, &success)
		if success == gl.FALSE {
			// var logLength int32
			// gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

			gl.GetShaderInfoLog(shader, 1024, nil, gl.Str(infoLog))

			return fmt.Errorf("Shader compilation error of type: %s\n %s", shaderType, infoLog)
		}
	} else {
		gl.GetProgramiv(shader, gl.LINK_STATUS, &success)
		if success == gl.FALSE {
			// var logLength int32
			// gl.GetProgramiv(shader, gl.INFO_LOG_LENGTH, &logLength)

			gl.GetProgramInfoLog(shader, 1024, nil, gl.Str(infoLog))

			return fmt.Errorf("Program linking error of type: %s\n %s", shaderType, infoLog)
		}
	}
	return nil
}
