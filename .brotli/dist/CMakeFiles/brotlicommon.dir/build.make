# CMAKE generated file: DO NOT EDIT!
# Generated by "Unix Makefiles" Generator, CMake Version 3.10

# Delete rule output on recipe failure.
.DELETE_ON_ERROR:


#=============================================================================
# Special targets provided by cmake.

# Disable implicit rules so canonical targets will work.
.SUFFIXES:


# Remove some rules from gmake that .SUFFIXES does not remove.
SUFFIXES =

.SUFFIXES: .hpux_make_needs_suffix_list


# Suppress display of executed commands.
$(VERBOSE).SILENT:


# A target that is always out of date.
cmake_force:

.PHONY : cmake_force

#=============================================================================
# Set environment variables for the build.

# The shell in which to execute make rules.
SHELL = /bin/sh

# The CMake executable.
CMAKE_COMMAND = /usr/bin/cmake

# The command to remove a file.
RM = /usr/bin/cmake -E remove -f

# Escaping for special characters.
EQUALS = =

# The top-level source directory on which CMake was run.
CMAKE_SOURCE_DIR = /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli

# The top-level build directory on which CMake was run.
CMAKE_BINARY_DIR = /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/dist

# Include any dependencies generated for this target.
include CMakeFiles/brotlicommon.dir/depend.make

# Include the progress variables for this target.
include CMakeFiles/brotlicommon.dir/progress.make

# Include the compile flags for this target's objects.
include CMakeFiles/brotlicommon.dir/flags.make

CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o: CMakeFiles/brotlicommon.dir/flags.make
CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o: ../c/common/dictionary.c
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green --progress-dir=/home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/dist/CMakeFiles --progress-num=$(CMAKE_PROGRESS_1) "Building C object CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o"
	/usr/bin/cc $(C_DEFINES) $(C_INCLUDES) $(C_FLAGS) -o CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o   -c /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/c/common/dictionary.c

CMakeFiles/brotlicommon.dir/c/common/dictionary.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/brotlicommon.dir/c/common/dictionary.c.i"
	/usr/bin/cc $(C_DEFINES) $(C_INCLUDES) $(C_FLAGS) -E /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/c/common/dictionary.c > CMakeFiles/brotlicommon.dir/c/common/dictionary.c.i

CMakeFiles/brotlicommon.dir/c/common/dictionary.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/brotlicommon.dir/c/common/dictionary.c.s"
	/usr/bin/cc $(C_DEFINES) $(C_INCLUDES) $(C_FLAGS) -S /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/c/common/dictionary.c -o CMakeFiles/brotlicommon.dir/c/common/dictionary.c.s

CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o.requires:

.PHONY : CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o.requires

CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o.provides: CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o.requires
	$(MAKE) -f CMakeFiles/brotlicommon.dir/build.make CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o.provides.build
.PHONY : CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o.provides

CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o.provides.build: CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o


CMakeFiles/brotlicommon.dir/c/common/transform.c.o: CMakeFiles/brotlicommon.dir/flags.make
CMakeFiles/brotlicommon.dir/c/common/transform.c.o: ../c/common/transform.c
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green --progress-dir=/home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/dist/CMakeFiles --progress-num=$(CMAKE_PROGRESS_2) "Building C object CMakeFiles/brotlicommon.dir/c/common/transform.c.o"
	/usr/bin/cc $(C_DEFINES) $(C_INCLUDES) $(C_FLAGS) -o CMakeFiles/brotlicommon.dir/c/common/transform.c.o   -c /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/c/common/transform.c

CMakeFiles/brotlicommon.dir/c/common/transform.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/brotlicommon.dir/c/common/transform.c.i"
	/usr/bin/cc $(C_DEFINES) $(C_INCLUDES) $(C_FLAGS) -E /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/c/common/transform.c > CMakeFiles/brotlicommon.dir/c/common/transform.c.i

CMakeFiles/brotlicommon.dir/c/common/transform.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/brotlicommon.dir/c/common/transform.c.s"
	/usr/bin/cc $(C_DEFINES) $(C_INCLUDES) $(C_FLAGS) -S /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/c/common/transform.c -o CMakeFiles/brotlicommon.dir/c/common/transform.c.s

CMakeFiles/brotlicommon.dir/c/common/transform.c.o.requires:

.PHONY : CMakeFiles/brotlicommon.dir/c/common/transform.c.o.requires

CMakeFiles/brotlicommon.dir/c/common/transform.c.o.provides: CMakeFiles/brotlicommon.dir/c/common/transform.c.o.requires
	$(MAKE) -f CMakeFiles/brotlicommon.dir/build.make CMakeFiles/brotlicommon.dir/c/common/transform.c.o.provides.build
.PHONY : CMakeFiles/brotlicommon.dir/c/common/transform.c.o.provides

CMakeFiles/brotlicommon.dir/c/common/transform.c.o.provides.build: CMakeFiles/brotlicommon.dir/c/common/transform.c.o


# Object files for target brotlicommon
brotlicommon_OBJECTS = \
"CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o" \
"CMakeFiles/brotlicommon.dir/c/common/transform.c.o"

# External object files for target brotlicommon
brotlicommon_EXTERNAL_OBJECTS =

libbrotlicommon.so.1.0.5: CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o
libbrotlicommon.so.1.0.5: CMakeFiles/brotlicommon.dir/c/common/transform.c.o
libbrotlicommon.so.1.0.5: CMakeFiles/brotlicommon.dir/build.make
libbrotlicommon.so.1.0.5: CMakeFiles/brotlicommon.dir/link.txt
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green --bold --progress-dir=/home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/dist/CMakeFiles --progress-num=$(CMAKE_PROGRESS_3) "Linking C shared library libbrotlicommon.so"
	$(CMAKE_COMMAND) -E cmake_link_script CMakeFiles/brotlicommon.dir/link.txt --verbose=$(VERBOSE)
	$(CMAKE_COMMAND) -E cmake_symlink_library libbrotlicommon.so.1.0.5 libbrotlicommon.so.1 libbrotlicommon.so

libbrotlicommon.so.1: libbrotlicommon.so.1.0.5
	@$(CMAKE_COMMAND) -E touch_nocreate libbrotlicommon.so.1

libbrotlicommon.so: libbrotlicommon.so.1.0.5
	@$(CMAKE_COMMAND) -E touch_nocreate libbrotlicommon.so

# Rule to build all files generated by this target.
CMakeFiles/brotlicommon.dir/build: libbrotlicommon.so

.PHONY : CMakeFiles/brotlicommon.dir/build

CMakeFiles/brotlicommon.dir/requires: CMakeFiles/brotlicommon.dir/c/common/dictionary.c.o.requires
CMakeFiles/brotlicommon.dir/requires: CMakeFiles/brotlicommon.dir/c/common/transform.c.o.requires

.PHONY : CMakeFiles/brotlicommon.dir/requires

CMakeFiles/brotlicommon.dir/clean:
	$(CMAKE_COMMAND) -P CMakeFiles/brotlicommon.dir/cmake_clean.cmake
.PHONY : CMakeFiles/brotlicommon.dir/clean

CMakeFiles/brotlicommon.dir/depend:
	cd /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/dist && $(CMAKE_COMMAND) -E cmake_depends "Unix Makefiles" /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/dist /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/dist /home/vladimirlesk/go/src/github.com/wal-g/wal-g/.brotli/dist/CMakeFiles/brotlicommon.dir/DependInfo.cmake --color=$(COLOR)
.PHONY : CMakeFiles/brotlicommon.dir/depend

