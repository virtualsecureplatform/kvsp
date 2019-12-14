cmake_minimum_required(VERSION 3.5)

# Defaults to CMAKE_BUILD_TYPE=Debug
if(NOT CMAKE_BUILD_TYPE AND NOT CMAKE_CONFIGURATION_TYPES)
  set(CMAKE_BUILD_TYPE Debug CACHE STRING "Choose the type of build." FORCE)
endif()

project(kvsp)

add_subdirectory(cahp-sim)
add_subdirectory(tfheutil)
add_subdirectory(tfhe/src)
add_subdirectory(Iyokan-L2)

set_target_properties(cahp-sim tfheutil iyokanl2
    PROPERTIES
    ARCHIVE_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/lib"
    LIBRARY_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/lib"
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)
