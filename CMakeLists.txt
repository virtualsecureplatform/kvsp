cmake_minimum_required(VERSION 3.5)

# Defaults to CMAKE_BUILD_TYPE=Debug
if(NOT CMAKE_BUILD_TYPE AND NOT CMAKE_CONFIGURATION_TYPES)
  set(CMAKE_BUILD_TYPE Debug CACHE STRING "Choose the type of build." FORCE)
endif()

set(IYOKANL2_ENABLE_TFHE on)

if(KVSP_ENABLE_CUDA)
    project(kvsp LANGUAGES CUDA CXX)
    find_package(CUDA)
    set(IYOKANL2_ENABLE_CUFHE on)
    add_subdirectory(cuFHE/cufhe)
else(KVSP_ENABLE_CUDA)
    project(kvsp)
    set(IYOKANL2_ENABLE_CUFHE off)
endif(KVSP_ENABLE_CUDA)

add_subdirectory(cahp-sim)
add_subdirectory(tfheutil)
add_subdirectory(tfhe/src)
add_subdirectory(Iyokan-L2)

if(KVSP_ENABLE_CUDA)
    set_target_properties(iyokanl2-cufhe
        PROPERTIES
        ARCHIVE_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/lib"
        LIBRARY_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/lib"
        RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
    )
endif(KVSP_ENABLE_CUDA)

set_target_properties(cahp-sim tfheutil iyokanl2
    PROPERTIES
    ARCHIVE_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/lib"
    LIBRARY_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/lib"
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)
