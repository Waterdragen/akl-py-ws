# syntax=docker/dockerfile:1.2

FROM nginx

# Install python and pip
RUN apt-get update && apt-get install -y python3 python3-pip

# Set working directory
WORKDIR .

# Copy the Python requirements files
COPY python/Pipfile python/Pipfile.lock .

# Install pipenv
RUN pip3 install pipenv

# Install Python dependencies within a virtual environment
RUN pipenv install --system --deploy --ignore-pipfile

# Create and activate a virtual environment
RUN python3 -m venv /venv
ENV PATH="/venv/bin:$PATH"

# Install python dependencies
RUN pip3 install --no-cache-dir -r requirements.txt

# Run python server
CMD ["python", "python/manage.py runserver"]

# Disable nginx welcome page
RUN mv etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.disabled

# Copy nginx conf file
COPY nginx.conf /etc/nginx/conf.d/nginx.conf

# Start nginx and start websockets
CMD service nginx start && ./startws.sh
