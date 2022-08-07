#version 410
layout (location = 0) in vec2 aPos;
layout (location = 1) in vec2 texPos;

uniform mat4 view;
out vec2 TexCoord;

void main()
{
    gl_Position = view * vec4(aPos, 0.0, 1.0);
    TexCoord = vec2(texPos.s, 1.0 - texPos.t); // reflect frame vertically
}
