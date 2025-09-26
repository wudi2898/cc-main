#!/usr/bin/env python3
"""
依赖验证脚本
检查所有必需的依赖是否正确安装
"""

import sys
import importlib
import subprocess

def check_python_version():
    """检查Python版本"""
    version = sys.version_info
    if version.major < 3 or (version.major == 3 and version.minor < 7):
        print(f"❌ Python版本过低: {version.major}.{version.minor}")
        print("   需要Python 3.7+")
        return False
    else:
        print(f"✅ Python版本: {version.major}.{version.minor}.{version.micro}")
        return True

def check_dependency(module_name, package_name=None):
    """检查单个依赖"""
    if package_name is None:
        package_name = module_name
    
    try:
        importlib.import_module(module_name)
        print(f"✅ {package_name}")
        return True
    except ImportError:
        print(f"❌ {package_name} - 未安装")
        return False

def check_core_dependencies():
    """检查核心依赖"""
    print("\n🔍 检查核心依赖...")
    
    core_deps = [
        ("flask", "flask"),
        ("flask_socketio", "flask-socketio"),
        ("psutil", "psutil"),
        ("socks", "PySocks"),
    ]
    
    all_ok = True
    for module, package in core_deps:
        if not check_dependency(module, package):
            all_ok = False
    
    return all_ok

def check_optional_dependencies():
    """检查可选依赖"""
    print("\n🔍 检查可选依赖...")
    
    optional_deps = [
        ("requests", "requests"),
        ("cryptography", "cryptography"),
        ("eventlet", "eventlet"),
        ("uvloop", "uvloop"),
        ("colorlog", "colorlog"),
        ("yaml", "pyyaml"),
        ("dateutil", "python-dateutil"),
        ("jsonschema", "jsonschema"),
    ]
    
    installed_count = 0
    for module, package in optional_deps:
        if check_dependency(module, package):
            installed_count += 1
    
    print(f"\n📊 可选依赖安装情况: {installed_count}/{len(optional_deps)}")
    return installed_count

def check_pip_packages():
    """使用pip检查已安装的包"""
    print("\n🔍 检查已安装的包...")
    
    try:
        result = subprocess.run([sys.executable, "-m", "pip", "list"], 
                              capture_output=True, text=True)
        if result.returncode == 0:
            packages = result.stdout
            print("✅ pip list 命令执行成功")
            
            # 检查关键包
            key_packages = ["flask", "psutil", "PySocks"]
            for package in key_packages:
                if package in packages:
                    print(f"✅ {package} 已安装")
                else:
                    print(f"❌ {package} 未安装")
        else:
            print("❌ pip list 命令执行失败")
            return False
    except Exception as e:
        print(f"❌ 检查pip包时出错: {e}")
        return False
    
    return True

def main():
    """主函数"""
    print("🚀 CC压测工具依赖验证")
    print("=" * 40)
    
    # 检查Python版本
    if not check_python_version():
        print("\n❌ Python版本检查失败")
        return 1
    
    # 检查核心依赖
    core_ok = check_core_dependencies()
    
    # 检查可选依赖
    optional_count = check_optional_dependencies()
    
    # 检查pip包
    pip_ok = check_pip_packages()
    
    print("\n" + "=" * 40)
    print("📋 验证结果")
    print("=" * 40)
    
    if core_ok:
        print("✅ 核心依赖: 全部安装")
    else:
        print("❌ 核心依赖: 部分缺失")
        print("   请运行: pip install -r requirements-minimal.txt")
    
    if optional_count > 0:
        print(f"✅ 可选依赖: {optional_count} 个已安装")
    else:
        print("⚠️  可选依赖: 未安装")
        print("   请运行: pip install -r requirements.txt")
    
    if pip_ok:
        print("✅ pip检查: 正常")
    else:
        print("❌ pip检查: 异常")
    
    if core_ok:
        print("\n🎉 核心功能可以正常使用！")
        return 0
    else:
        print("\n❌ 请先安装核心依赖")
        return 1

if __name__ == "__main__":
    sys.exit(main())
