#version 450

layout(location=0) in vec4 fragColor;
// layout(location=1) in vec2 fragTexCoord;

// layout(binding=1) uniform sampler2D texSampler;

layout(location=0) out vec4 outColor;


void main() {
    // outColor = vec4(gl_FragCoord.z/gl_FragCoord.w, gl_FragCoord.z/gl_FragCoord.w, gl_FragCoord.z/gl_FragCoord.w, 1.0);
    // outColor = vec4(fragTexCoord, 0.0, 1.0);
    // outColor = texture(texSampler, fragTexCoord);
    outColor = fragColor;
}