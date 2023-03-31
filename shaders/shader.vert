#version 450

// layout(binding=0) uniform UniformBufferObject {
//     mat4 model;
//     mat4 view;
//     mat4 proj;
// } ubo;

layout(location=0) in vec3 inPosition;
layout(location=1) in vec3 inNormal;

// layout(location=1) in vec3 inColor;
// layout(location=2) in vec2 inTexCoord;


layout (push_constant) uniform constants {
    mat4 proj;
    mat4 model;
} pc;

layout(location=0) out vec4 fragColor;
// layout(location=1) out vec3 normal;
// layout(location=1) out vec2 fragTexCoord;

// Static light at (3,1,5)
// Phong Diffuse Lighting: diffuse component x dot(normal, light vec)
void main() {
    
    gl_Position = pc.proj*pc.model*vec4(inPosition, 1.0);
    
    // Static light
    vec3 lightVec = normalize(vec3(500,300,500) - inPosition);
    float ndotl = dot(inNormal, lightVec);

    vec4 baseColor = vec4(0.5,0.5,0.5,1); //vec4(abs(inNormal),1.0);
    fragColor = baseColor * ndotl;

    // fragTexCoord = inTexCoord;
}