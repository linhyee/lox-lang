#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
  #include <windows.h>
  #include <direct.h>
  #define MAX_PATH_LEN MAX_PATH
#else
  #include <unistd.h>
  #include <libgen.h>
  #define MAX_PATH_LEN 4096
#endif

#include "common.h"
#include "scanner.h"

#define MAX_DEPS 256
#define MAX_VISITED 256

typedef struct {
  char path[MAX_PATH_LEN];
} Dependency;

typedef struct {
  char path[MAX_PATH_LEN];
  Dependency deps[MAX_DEPS];
  int depCount;
} FileNode;

// 全局已访问文件列表（用于循环检测）
static int visitedCount = 0;
static char visited[MAX_VISITED][MAX_PATH_LEN];

// BFS 待处理队列
static char toProcess[MAX_VISITED][MAX_PATH_LEN];
static int toProcessCount;

// 将文件路径添加到已访问列表
static void addVisited(const char* path) {
  if (visitedCount < MAX_VISITED) {
    strncpy(visited[visitedCount], path, MAX_PATH_LEN - 1);
    visited[visitedCount][MAX_PATH_LEN - 1] = '\0';
    visitedCount++;
  }
}

// 检查文件是否已被访问
static int isVisited(const char* path) {
  for (int i = 0; i < visitedCount; i++) {
    if (strcmp(visited[i], path) == 0) {
      return 1;
    }
  }
  return 0;
}

// 清空已访问列表
static void clearVisited(void) {
  visitedCount = 0;
}

// 读取文件内容
static char* readFile(const char* path) {
  FILE* file = fopen(path, "rb");
  if (file == NULL) return NULL;

  fseek(file, 0L, SEEK_END);
  size_t fileSize = ftell(file);
  rewind(file);

  char* buffer = (char*)malloc(fileSize + 1);
  if (buffer == NULL) {
    fclose(file);
    return NULL;
  }

  size_t bytesRead = fread(buffer, sizeof(char), fileSize, file);
  fclose(file);
  
  if (bytesRead < fileSize) {
    free(buffer);
    return NULL;
  }

  buffer[bytesRead] = '\0';
  return buffer;
}

// 拼接路径（使用正斜杠）
static int joinPath(const char* baseDir, const char* relPath, char* out) {
  size_t baseLen = strlen(baseDir);
  size_t relLen = strlen(relPath);
  if (baseLen + 1 + relLen >= MAX_PATH_LEN) {
    return 0;
  }

  memcpy(out, baseDir, baseLen);
  out[baseLen] = '/';
  memcpy(out + baseLen + 1, relPath, relLen + 1);
  return 1;
}

// 将相对路径解析为绝对路径（跨平台）
static void resolvePath(const char* basePath, const char* relPath, char* out) {
  char baseDir[MAX_PATH_LEN];
  strncpy(baseDir, basePath, MAX_PATH_LEN - 1);
  baseDir[MAX_PATH_LEN - 1] = '\0';
  
  // 获取文件所在目录（去掉文件名部分）
  char* lastSlash = strrchr(baseDir, '/');
  char* lastBackslash = strrchr(baseDir, '\\');
  char* separator = (lastSlash > lastBackslash) ? lastSlash : lastBackslash;
  
  if (separator != NULL) {
    *separator = '\0';  // 截断到目录部分
  } else {
    strcpy(baseDir, ".");
  }

  // 构建完整路径
#ifdef _WIN32
  char resolved[MAX_PATH_LEN];
  if (!joinPath(baseDir, relPath, resolved)) {
    strncpy(out, relPath, MAX_PATH_LEN - 1);
    out[MAX_PATH_LEN - 1] = '\0';
    return;
  }
  char normalized[MAX_PATH_LEN];
  if (_fullpath(normalized, resolved, MAX_PATH_LEN) == NULL) {
    strncpy(out, resolved, MAX_PATH_LEN - 1);
  } else {
    strncpy(out, normalized, MAX_PATH_LEN - 1);
  }
#else
  char resolved[MAX_PATH_LEN];
  if (!joinPath(baseDir, relPath, resolved)) {
    strncpy(out, relPath, MAX_PATH_LEN - 1);
    out[MAX_PATH_LEN - 1] = '\0';
    return;
  }
  char* realPath = realpath(resolved, NULL);
  if (realPath != NULL) {
    strncpy(out, realPath, MAX_PATH_LEN - 1);
    free(realPath);
  } else {
    strncpy(out, resolved, MAX_PATH_LEN - 1);
  }
#endif
  
  out[MAX_PATH_LEN - 1] = '\0';
  
  // 统一将反斜杠转换为正斜杠
  for (char* p = out; *p != '\0'; p++) {
    if (*p == '\\') *p = '/';
  }
}

// 从文件中提取所有的 import 语句
static void extractImports(const char* filePath, FileNode* node) {
  char* source = readFile(filePath);
  if (source == NULL) {
    fprintf(stderr, "Warning: Could not read file: %s\n", filePath);
    return;
  }

  initScanner(source);
  Token token;
  
  do {
    token = scanToken();
    if (token.type == TOKEN_IMPORT) {
      token = scanToken();
      if (token.type == TOKEN_STRING) {
        // 提取字符串中的路径（去掉引号）
        char importPath[MAX_PATH_LEN];
        int len = token.length - 2;
        if (len > 0 && len < MAX_PATH_LEN - 1) {
          strncpy(importPath, token.start + 1, len);
          importPath[len] = '\0';
          
          // 解析为绝对路径
          char resolvedPath[MAX_PATH_LEN];
          resolvePath(filePath, importPath, resolvedPath);
          
          // 如果没有 .lox 扩展名，自动添加
          size_t lenPath = strlen(resolvedPath);
          if (lenPath < 4 || strcmp(resolvedPath + lenPath - 4, ".lox") != 0) {
            if (lenPath + 5 < MAX_PATH_LEN) {
              strncat(resolvedPath, ".lox", MAX_PATH_LEN - lenPath - 1);
            }
          }
          
          // 添加到依赖列表
          if (node->depCount < MAX_DEPS) {
            strncpy(node->deps[node->depCount].path, resolvedPath, MAX_PATH_LEN - 1);
            node->deps[node->depCount].path[MAX_PATH_LEN - 1] = '\0';
            node->depCount++;
          }
        }
      }
    }
  } while (token.type != TOKEN_EOF);

  free(source);
}

// 打印依赖树（迭代式 DFS，避免递归栈溢出）
static void printTreeIterative(FileNode* nodes, int nodeCount, const char* rootPath) {
  typedef struct {
    int nodeIdx;                    // 节点索引
    int depth;                      // 深度层级
    int nextChildIndex;             // 下一个要处理的子节点索引
    int parentIndices[MAX_VISITED]; // 祖先节点索引列表（用于循环检测）
    int parentCount;                // 祖先节点数量
  } StackFrame;
  
  StackFrame* stack = (StackFrame*)malloc(MAX_VISITED * sizeof(StackFrame));
  if (!stack) {
    fprintf(stderr, "Memory allocation failed\n");
    return;
  }
  
  int stackTop = 0;
  
  // 查找根节点索引
  int rootIdx = -1;
  for (int i = 0; i < nodeCount; i++) {
    if (strcmp(nodes[i].path, rootPath) == 0) {
      rootIdx = i;
      break;
    }
  }
  
  if (rootIdx == -1) {
    free(stack);
    return;
  }
  
  // 初始化根节点栈帧
  stack[0].nodeIdx = rootIdx;
  stack[0].depth = 0;
  stack[0].nextChildIndex = 0;
  stack[0].parentCount = 0;
  stackTop = 1;
  
  while (stackTop > 0) {
    StackFrame* frame = &stack[stackTop - 1];
    FileNode* node = &nodes[frame->nodeIdx];
    
    // 首次访问该节点时打印节点信息
    if (frame->nextChildIndex == 0) {
      char indent[256] = "";
      for (int i = 0; i < frame->depth; i++) {
        strcat(indent, "  ");
      }
      
      // 获取文件名（不包含路径）
      const char* filename = node->path;
      const char* lastSlash = strrchr(node->path, '/');
      if (lastSlash != NULL) {
        filename = lastSlash + 1;
      }
      
      printf("%s+-- %s\n", indent, filename);
      
      // 检测自引用循环（文件导入自身）
      for (int i = 0; i < frame->parentCount; i++) {
        if (frame->parentIndices[i] == frame->nodeIdx) {
          printf("%s  [CIRCULAR IMPORT DETECTED]\n", indent);
          stackTop--;
          continue;
        }
      }
    }
    
    // 处理子节点
    if (frame->nextChildIndex < node->depCount) {
      int childIdx = -1;
      const char* childPath = node->deps[frame->nextChildIndex].path;
      
      // 查找子节点索引
      for (int i = 0; i < nodeCount; i++) {
        if (strcmp(nodes[i].path, childPath) == 0) {
          childIdx = i;
          break;
        }
      }
      
      frame->nextChildIndex++;
      
      if (childIdx != -1) {
        // 检测循环依赖
        int isCircular = 0;
        for (int i = 0; i < frame->parentCount; i++) {
          if (frame->parentIndices[i] == childIdx) {
            isCircular = 1;
            break;
          }
        }
        
        if (isCircular) {
          // 打印循环依赖标记
          char indent[256] = "";
          for (int i = 0; i < frame->depth + 1; i++) {
            strcat(indent, "  ");
          }
          const char* filename = childPath;
          const char* lastSlash = strrchr(childPath, '/');
          if (lastSlash != NULL) {
            filename = lastSlash + 1;
          }
          printf("%s+-- %s [CIRCULAR]\n", indent, filename);
        } else {
          // 将子节点压入栈中继续处理
          if (stackTop < MAX_VISITED) {
            StackFrame* newFrame = &stack[stackTop];
            newFrame->nodeIdx = childIdx;
            newFrame->depth = frame->depth + 1;
            newFrame->nextChildIndex = 0;
            // 复制祖先节点列表
            for (int i = 0; i < frame->parentCount; i++) {
              newFrame->parentIndices[i] = frame->parentIndices[i];
            }
            newFrame->parentIndices[frame->parentCount] = frame->nodeIdx;
            newFrame->parentCount = frame->parentCount + 1;
            stackTop++;
          }
        }
      }
    } else {
      // 所有子节点处理完毕，出栈
      stackTop--;
    }
  }
  
  free(stack);
}

// 打印使用说明
static void printUsage(const char* progName) {
  fprintf(stderr, "usage: %s <lox-file>\n", progName);
  fprintf(stderr, "  Display import dependency tree for a .lox file\n");
}

int main(int argc, const char* argv[]) {
  if (argc != 2) {
    printUsage(argv[0]);
    return 1;
  }

  const char* filePath = argv[1];
  
  // 规范化输入路径为绝对路径
  char* normalized = (char*)malloc(MAX_PATH_LEN);
  if (!normalized) {
    fprintf(stderr, "Memory allocation failed\n");
    return 1;
  }
  
#ifdef _WIN32
  if (_fullpath(normalized, filePath, MAX_PATH_LEN) == NULL) {
    strncpy(normalized, filePath, MAX_PATH_LEN - 1);
    normalized[MAX_PATH_LEN - 1] = '\0';
  }
#else
  char* realPath = realpath(filePath, NULL);
  if (realPath != NULL) {
    strncpy(normalized, realPath, MAX_PATH_LEN - 1);
    normalized[MAX_PATH_LEN - 1] = '\0';
    free(realPath);
  } else {
    strncpy(normalized, filePath, MAX_PATH_LEN - 1);
    normalized[MAX_PATH_LEN - 1] = '\0';
  }
#endif
  
  // 统一分隔符为正斜杠
  for (char* p = normalized; *p != '\0'; p++) {
    if (*p == '\\') *p = '/';
  }
  
  printf("Import Dependency Tree for: %s\n\n", normalized);
  
  clearVisited();
  
  // 分配节点数组
  FileNode* nodes = (FileNode*)malloc(MAX_VISITED * sizeof(FileNode));
  if (!nodes) {
    fprintf(stderr, "Memory allocation failed\n");
    free(normalized);
    return 1;
  }
  
  int nodeCount = 0;
  
  // BFS 初始化：从根文件开始
  toProcessCount = 1;
  strncpy(toProcess[0], normalized, MAX_PATH_LEN - 1);
  toProcess[0][MAX_PATH_LEN - 1] = '\0';
  
  // BFS 遍历所有依赖文件
  while (toProcessCount > 0) {
    char current[MAX_PATH_LEN];
    strcpy(current, toProcess[0]);
    
    // 队列出队
    for (int i = 0; i < toProcessCount - 1; i++) {
      strcpy(toProcess[i], toProcess[i + 1]);
    }
    toProcessCount--;
    
    // 跳过已访问的文件
    if (isVisited(current)) continue;
    addVisited(current);
    
    if (nodeCount >= MAX_VISITED) {
      fprintf(stderr, "Warning: Too many files, stopping at %d\n", MAX_VISITED);
      break;
    }
    
    // 创建文件节点并提取其依赖
    FileNode* node = &nodes[nodeCount];
    strncpy(node->path, current, MAX_PATH_LEN - 1);
    node->path[MAX_PATH_LEN - 1] = '\0';
    node->depCount = 0;
    
    extractImports(current, node);
    nodeCount++;
    
    // 将未访问的依赖加入待处理队列
    for (int i = 0; i < node->depCount; i++) {
      if (!isVisited(node->deps[i].path)) {
        if (toProcessCount < MAX_VISITED) {
          strcpy(toProcess[toProcessCount], node->deps[i].path);
          toProcessCount++;
        }
      }
    }
  }
  
  printf("\n");
  // 打印依赖树（使用 DFS 方式）
  printTreeIterative(nodes, nodeCount, normalized);
  
  free(nodes);
  free(normalized);
  
  return 0;
}