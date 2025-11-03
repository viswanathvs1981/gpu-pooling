"""
NexusAI Python SDK Setup
"""

from setuptools import setup, find_packages

setup(
    name="nexusai-sdk",
    version="1.0.0",
    description="NexusAI Platform SDK for Microsoft Agent Framework Integration",
    author="NexusAI Team",
    packages=find_packages(),
    install_requires=[
        "httpx>=0.25.0",
        "agent-framework>=1.0.0b251028",
    ],
    python_requires=">=3.9",
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
    ],
)

