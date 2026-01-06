#!/usr/bin/env python3
"""
Python AST Helper for Katichai
Extracts functions, classes, and imports from Python files using AST
Outputs JSON matching Go's FileAnalysis structure
"""

import ast
import json
import sys


class ComplexityVisitor(ast.NodeVisitor):
    """Calculate cyclomatic complexity of a function"""
    def __init__(self):
        self.complexity = 1  # Base complexity
    
    def visit_If(self, node):
        self.complexity += 1
        self.generic_visit(node)
    
    def visit_For(self, node):
        self.complexity += 1
        self.generic_visit(node)
    
    def visit_While(self, node):
        self.complexity += 1
        self.generic_visit(node)
    
    def visit_And(self, node):
        self.complexity += 1
        self.generic_visit(node)
    
    def visit_Or(self, node):
        self.complexity += 1
        self.generic_visit(node)
    
    def visit_ExceptHandler(self, node):
        self.complexity += 1
        self.generic_visit(node)
    
    def visit_With(self, node):
        self.complexity += 1
        self.generic_visit(node)
    
    def visit_Try(self, node):
        self.complexity += 1
        self.generic_visit(node)


def get_docstring(node):
    """Extract docstring from a node, truncated if too long"""
    if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
        docstring = ast.get_docstring(node)
        if docstring:
            # Truncate very long docstrings to reduce token usage (keep first 200 chars)
            if len(docstring) > 200:
                return docstring[:200] + "... [truncated]"
            return docstring
    return ""


def get_function_body(node, source_lines):
    """Extract function body source code including decorators (decorators are important context)"""
    # Include decorators as they provide important context (e.g., @property, @staticmethod)
    # Token savings come from docstring truncation and diff context reduction instead
    if hasattr(node, 'decorator_list') and node.decorator_list:
        start = node.decorator_list[0].lineno - 1  # 0-indexed, start from first decorator
    else:
        start = node.lineno - 1  # 0-indexed
    
    end = node.end_lineno if node.end_lineno else node.lineno
    return '\n'.join(source_lines[start:end])


def calculate_complexity(node):
    """Calculate cyclomatic complexity of a function"""
    visitor = ComplexityVisitor()
    visitor.visit(node)
    return visitor.complexity


def extract_parameters(node):
    """Extract function parameters"""
    params = []
    if hasattr(node, 'args'):
        args = node.args
        # Regular arguments
        for arg in args.args:
            params.append(arg.arg)
        # *args
        if args.vararg:
            params.append(f"*{args.vararg.arg}")
        # **kwargs
        if args.kwarg:
            params.append(f"**{args.kwarg.arg}")
    return params


def extract_return_type(node):
    """Extract return type annotation if present"""
    if hasattr(node, 'returns') and node.returns:
        try:
            return ast.unparse(node.returns)
        except:
            return ""
    return ""


def is_exported(name):
    """Check if name is exported (doesn't start with underscore)"""
    return not name.startswith('_')


def extract_function_info(node, source_lines):
    """Extract information from a function definition"""
    # Calculate LOC (includes decorators in body for context, but LOC metric is function-only)
    func_start = node.lineno
    func_end = node.end_lineno if node.end_lineno else node.lineno
    loc = func_end - func_start + 1
    
    return {
        "name": node.name,
        "start_line": node.lineno,
        "end_line": func_end,
        "loc": loc,
        "complexity": calculate_complexity(node),
        "parameters": extract_parameters(node),
        "return_type": extract_return_type(node),
        "is_exported": is_exported(node.name),
        "comments": get_docstring(node),  # Truncated to 200 chars to reduce tokens
        "body": get_function_body(node, source_lines)  # Includes decorators for context
    }


def extract_class_fields(node):
    """Extract class fields from assignments in __init__ or class body"""
    fields = []
    seen = set()
    
    for item in node.body:
        # Class-level assignments
        if isinstance(item, ast.AnnAssign) and isinstance(item.target, ast.Name):
            field_name = item.target.id
            if field_name not in seen:
                field_type = ""
                if item.annotation:
                    try:
                        field_type = ast.unparse(item.annotation)
                    except:
                        pass
                fields.append({"name": field_name, "type": field_type})
                seen.add(field_name)
        
        # Look for self.x assignments in __init__
        if isinstance(item, ast.FunctionDef) and item.name == "__init__":
            for stmt in ast.walk(item):
                if isinstance(stmt, ast.Assign):
                    for target in stmt.targets:
                        if isinstance(target, ast.Attribute) and isinstance(target.value, ast.Name):
                            if target.value.id == "self":
                                field_name = target.attr
                                if field_name not in seen:
                                    fields.append({"name": field_name, "type": ""})
                                    seen.add(field_name)
    
    return fields


def extract_class_info(node, source_lines):
    """Extract information from a class definition"""
    methods = []
    
    # Extract methods from class body
    for item in node.body:
        if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)):
            methods.append(extract_function_info(item, source_lines))
    
    return {
        "name": node.name,
        "start_line": node.lineno,
        "end_line": node.end_lineno if node.end_lineno else node.lineno,
        "methods": methods,
        "fields": extract_class_fields(node),
        "is_exported": is_exported(node.name),
        "comments": get_docstring(node)
    }


def extract_imports(node):
    """Extract import information"""
    imports = []
    
    if isinstance(node, ast.Import):
        for alias in node.names:
            imports.append({
                "path": alias.name,
                "alias": alias.asname if alias.asname else ""
            })
    
    elif isinstance(node, ast.ImportFrom):
        module = node.module if node.module else ""
        for alias in node.names:
            # For "from x import y", path is "x.y" or just "x" for "from x import *"
            if alias.name == "*":
                path = module
            else:
                path = f"{module}.{alias.name}" if module else alias.name
            
            imports.append({
                "path": path,
                "alias": alias.asname if alias.asname else ""
            })
    
    return imports


def parse_python_file(file_path):
    """Parse a Python file and extract AST information"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            source_code = f.read()
        
        source_lines = source_code.splitlines()
        tree = ast.parse(source_code, filename=file_path)
        
        functions = []
        classes = []
        imports = []
        
        # Walk the AST and extract information
        for node in ast.walk(tree):
            # Extract top-level functions (not methods inside classes)
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                # Check if this is a top-level function (not a method)
                # We'll add methods separately when processing classes
                is_method = False
                for parent in ast.walk(tree):
                    if isinstance(parent, ast.ClassDef):
                        if node in parent.body:
                            is_method = True
                            break
                
                if not is_method:
                    functions.append(extract_function_info(node, source_lines))
            
            # Extract classes
            elif isinstance(node, ast.ClassDef):
                classes.append(extract_class_info(node, source_lines))
            
            # Extract imports
            elif isinstance(node, (ast.Import, ast.ImportFrom)):
                imports.extend(extract_imports(node))
        
        # Output JSON
        result = {
            "functions": functions,
            "classes": classes,
            "imports": imports
        }
        
        print(json.dumps(result, indent=2))
        return 0
    
    except SyntaxError as e:
        # Syntax error in Python file
        print(f"Syntax error: {e}", file=sys.stderr)
        return 1
    except Exception as e:
        # Other errors
        print(f"Error: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python_ast_helper.py <file_path>", file=sys.stderr)
        sys.exit(1)
    
    file_path = sys.argv[1]
    sys.exit(parse_python_file(file_path))
