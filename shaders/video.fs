#version 410
out vec4 FragmentColor;

in vec2 TexCoord;
uniform sampler2D texImage;

void main() {
  FragmentColor = texture(texImage, TexCoord);
}
