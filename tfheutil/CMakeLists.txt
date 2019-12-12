cmake_minimum_required(VERSION 3.5)

# Defaults to CMAKE_BUILD_TYPE=Debug
if(NOT CMAKE_BUILD_TYPE AND NOT CMAKE_CONFIGURATION_TYPES)
  set(CMAKE_BUILD_TYPE Debug CACHE STRING "Choose the type of build." FORCE)
endif()

project(tfheutil)

add_executable(tfheutil main.cpp)
set_target_properties(tfheutil PROPERTIES
    CXX_STANDARD 17
    CXX_STANDARD_REQUIRED YES
    CXX_EXTENSIONS NO
)
target_compile_options(tfheutil
    PRIVATE -Wall -Wextra
    PRIVATE $<$<CONFIG:DEBUG>:-O0 -g3>
    PRIVATE $<$<CONFIG:RELEASE>:-O2>)
target_link_libraries(tfheutil
    PRIVATE tfhe-spqlios-avx
)
target_sources(tfheutil
    PRIVATE
        main.cpp
    )
