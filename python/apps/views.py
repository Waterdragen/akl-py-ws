"""
Django request handler
"""
from django.shortcuts import render
from django.http.request import HttpRequest

def bad_request(request: HttpRequest, exception: Exception):
    return render(request, "bad_request.html", {})
