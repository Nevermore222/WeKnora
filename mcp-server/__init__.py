#!/usr/bin/env python3
"""
Xelora MCP Server Package

A Model Context Protocol server that provides access to the Xelora knowledge management API.
"""

__version__ = "1.0.0"
__author__ = "Xelora Team"
__description__ = "Xelora MCP Server - Model Context Protocol server for Xelora API"

from .xelora_mcp_server import XeloraClient, run

__all__ = ["XeloraClient", "run"]
