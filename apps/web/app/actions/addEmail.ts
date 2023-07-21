'use server'
export async function addEmail(formData : FormData) {
    const email = formData.get('email');
    
    const res = await fetch('https://app.loops.so/api/newsletter-form/clk4gj1jq00myl70o5gv88fnf',{
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({email: email})
    })
    const json = await res.json();
    return json;
  }