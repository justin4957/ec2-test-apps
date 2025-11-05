FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY satirical_fix_generator.py .

# Expose port
EXPOSE 7070

# Set environment variable for port (can be overridden)
ENV PORT=7070

# Run the application
CMD ["python3", "satirical_fix_generator.py"]
