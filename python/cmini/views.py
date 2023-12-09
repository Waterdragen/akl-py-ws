"""
Django request handler
"""
from django.shortcuts import render
from django.http.request import HttpRequest
from django.http import HttpResponse

def bad_request(request: HttpRequest):
    return render(request, "bad_request.html", status=400)
