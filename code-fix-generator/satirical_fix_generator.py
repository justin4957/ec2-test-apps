#!/usr/bin/env python3
"""
Satirical Code Fix Generator

Uses DeepSeek Coder to generate absurd, satirical "fixes" for errors.
Integrates with the error-generator and slogan-server for maximum absurdity.
"""

import os
import logging
from flask import Flask, request, jsonify
from openai import OpenAI

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Configuration
DEEPSEEK_API_KEY = os.getenv('DEEPSEEK_API_KEY', '')
PORT = int(os.getenv('PORT', '7070'))

# Initialize Flask app
app = Flask(__name__)

# Initialize DeepSeek client
deepseek_client = None
if DEEPSEEK_API_KEY:
    deepseek_client = OpenAI(
        api_key=DEEPSEEK_API_KEY,
        base_url="https://api.deepseek.com"
    )
    logger.info("✓ DeepSeek client initialized")
else:
    logger.warning("⚠️  DEEPSEEK_API_KEY not set - using fallback responses")


SATIRICAL_SYSTEM_PROMPT = """You are a satirical software engineer who writes absurdly over-engineered,
philosophically profound, and intentionally ridiculous "fixes" for software errors.

Your fixes should:
1. Be technically valid code (Python or JavaScript)
2. Include excessive comments with existential observations
3. Use unnecessarily complex patterns (factory of factories, quantum observers, etc.)
4. Reference the slogan provided
5. Be 10-30 lines of actual code
6. Be funny and absurd while technically functional

Style: Combine Silicon Valley tech bro culture, philosophy, and complete overkill.
"""


FALLBACK_FIXES = [
    '''def fix_null_pointer():
    """
    Fix NullPointer by simply refusing to acknowledge null exists.
    Inspired by: {slogan}
    """
    class SchrodingerPointer:
        def __init__(self):
            self.value = None  # Or is it?

        def __getattribute__(self, name):
            # If we don't observe it, it's both null and not null
            if name == 'value':
                import random
                return random.choice([None, "probably_fine", 42])
            return super().__getattribute__(name)

    return SchrodingerPointer()  # Problem solved!
''',

    '''async function fixTimeout() {
    /*
     * Fix timeout by implementing quantum time dilation
     * Based on wisdom: {slogan}
     */
    const TIME_UNCERTAINTY = 0.5;  // Heisenberg would approve

    async function waitWithExistentialDread(ms) {
        console.log("⏳ Waiting... but what IS time, really?");

        // If time is relative, maybe the timeout already happened?
        const schrodingerTime = Math.random() > TIME_UNCERTAINTY
            ? 0  // Already done in another timeline
            : ms * 1000;  // Make it longer, just to be sure

        await new Promise(resolve => setTimeout(resolve, schrodingerTime));
    }

    return await waitWithExistentialDread(1);
}
''',

    '''class HeapOverflowHandler:
    """
    Handle heap overflow with a factory pattern wrapped in observers.
    Philosophy: {slogan}
    """

    def __init__(self):
        self.overflow_dimension = []  # Extra-dimensional storage

    def handle_overflow(self, data):
        # When the heap overflows, just put it somewhere else
        # This is called "problem deflection" in academic circles

        if len(self.overflow_dimension) > 1000000:
            # If extra dimension overflows, create another one
            self.overflow_dimension = [self.overflow_dimension]
            print("🌌 Created nested dimension for overflow")

        self.overflow_dimension.append(data)
        return True  # It's handled (somewhere)

    def get_data(self, index):
        # Good luck finding it now!
        return "¯\\_(ツ)_/¯"
'''
]


def generate_satirical_fix(error_message: str, slogan: str, error_type: str = "basic") -> str:
    """
    Generate a satirical code fix using DeepSeek Coder.

    Args:
        error_message: The error that needs "fixing"
        slogan: The sardonic slogan from slogan-server
        error_type: Type of error (basic, business, chaotic, philosophical)

    Returns:
        Satirical code that "fixes" the error
    """

    if not deepseek_client:
        # Use fallback
        import random
        fix = random.choice(FALLBACK_FIXES)
        return fix.format(slogan=slogan)

    # Adjust tone based on error type
    tone_modifiers = {
        "basic": "straightforward but over-engineered",
        "business": "corporate buzzword-heavy with synergy",
        "chaotic": "increasingly unhinged and complex",
        "philosophical": "deeply philosophical and existential"
    }

    tone = tone_modifiers.get(error_type, "absurdly over-engineered")

    user_prompt = f"""Generate a {tone} satirical code fix for this error:

Error: {error_message}
Slogan: {slogan}

Write actual code (Python or JavaScript) that "solves" this error in the most ridiculous way possible.
Include inline comments referencing the slogan.
Make it 15-30 lines of functional but absurd code.
"""

    try:
        response = deepseek_client.chat.completions.create(
            model="deepseek-coder",
            messages=[
                {"role": "system", "content": SATIRICAL_SYSTEM_PROMPT},
                {"role": "user", "content": user_prompt}
            ],
            temperature=0.9,  # High creativity
            max_tokens=800
        )

        code_fix = response.choices[0].message.content

        # Clean up markdown code blocks if present
        if "```" in code_fix:
            # Extract code from markdown blocks
            import re
            code_blocks = re.findall(r'```(?:python|javascript)?\n(.*?)```', code_fix, re.DOTALL)
            if code_blocks:
                code_fix = code_blocks[0]

        return code_fix.strip()

    except Exception as e:
        logger.error(f"DeepSeek API error: {e}")
        # Fallback
        import random
        fix = random.choice(FALLBACK_FIXES)
        return fix.format(slogan=slogan)


@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'satirical-fix-generator',
        'deepseek_available': deepseek_client is not None
    })


@app.route('/api/generate-fix', methods=['POST'])
def generate_fix():
    """
    Generate satirical fix for an error.

    Request:
    {
        "error": "NullPointerException in UserService.java:42",
        "slogan": "Off by one: Close enough is good enough",
        "error_type": "basic"
    }

    Response:
    {
        "success": true,
        "fix": "def handle_null():\n    ..."
    }
    """
    try:
        data = request.json
        error = data.get('error', '')
        slogan = data.get('slogan', '')
        error_type = data.get('error_type', 'basic')

        if not error or not slogan:
            return jsonify({
                'success': False,
                'error': 'Missing error or slogan'
            }), 400

        logger.info(f"Generating fix for: {error[:50]}...")

        fix = generate_satirical_fix(error, slogan, error_type)

        logger.info(f"✓ Generated {len(fix)} character fix")

        return jsonify({
            'success': True,
            'fix': fix,
            'error': error,
            'slogan': slogan
        })

    except Exception as e:
        logger.error(f"Error generating fix: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


def main():
    """Run the satirical fix generator service"""
    logger.info("🤖 Satirical Code Fix Generator")
    logger.info("=" * 60)

    if DEEPSEEK_API_KEY:
        logger.info("✓ DeepSeek API key configured")
    else:
        logger.warning("⚠️  No DeepSeek API key - using fallback fixes")
        logger.warning("   Set DEEPSEEK_API_KEY environment variable")

    logger.info(f"Starting server on port {PORT}...")
    logger.info("=" * 60)

    app.run(host='0.0.0.0', port=PORT, debug=False)


if __name__ == '__main__':
    main()
